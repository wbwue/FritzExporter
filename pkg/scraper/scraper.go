package scraper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/wbwue/FritzExporter/pkg/config"
	"github.com/wbwue/FritzExporter/pkg/fritz"
	"github.com/wbwue/FritzExporter/pkg/loki"

	"github.com/ndecker/fritzbox_exporter/fritzbox_upnp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	loginSid    *string
	lastLogTime *time.Time

	LanDevicesOnline = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_online",
		Help: "Gauge showing online state of device",
	}, []string{"name", "ip", "mac", "dev_type"})
	LanDevicesActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_active",
		Help: "Gauge showing active state of device",
	}, []string{"name", "ip", "mac", "dev_type"})
	LanDevicesSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_speed",
		Help: "Gauge showing speed of device",
	}, []string{"name", "ip", "mac", "dev_type"})
	WlanDeviceSignal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_wlan_devices_signal",
		Help: "Gauge showing signal strength of wifi devices",
	}, []string{"name", "ip", "mac", "dev_type"})
	WlanDeviceSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_wlan_devices_speed",
		Help: "Gauge showing current speed of wifi devices",
	}, []string{"name", "ip", "mac", "dev_type", "direction"})
	WlanDeviceSpeedMax = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_wlan_devices_speed_max",
		Help: "Gauge showing maximum speed of wifi devices",
	}, []string{"name", "ip", "mac", "dev_type", "direction"})
	WlanDeviceInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_wlan_devices_info",
		Help: "Gauge showing maximum speed of wifi devices",
	}, []string{"name", "ip", "mac", "dev_type", "band", "standard", "encryption"})

	InternetDownstreamSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_internet_downstream_current",
		Help: "Gauge showing latest internet downstream speed",
	}, []string{"type"})
	InternetUpstreamSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_internet_upstream_current",
		Help: "Gauge showing latest internet upstream speed",
	}, []string{"type"})
)

type Scraper struct {
	cfg              *config.Config
	logger           log.Logger
	deviceband       map[string]prometheus.Labels
	upnpServicesRoot *fritzbox_upnp.Root
	connectionInfos  *prometheus.Labels
	logPusher        loki.Pusher
}

func NewScraper(config *config.Config, logger log.Logger) *Scraper {
	return &Scraper{
		cfg:        config,
		logger:     logger,
		deviceband: make(map[string]prometheus.Labels),
		logPusher:  loki.New(config.LokiURL),
	}
}

func (s *Scraper) Run(ctx context.Context) error {
	zeroTime := time.Unix(0, 0)
	lastLogTime = &zeroTime

	if s.cfg.LogPath != "" {
		f, err := os.Create(s.cfg.LogPath)
		if err != nil {
			level.Warn(s.logger).Log("Failed to open log file")
		} else {
			f.Close()
		}
	}
	err := s.loadServices()
	if err != nil {
		level.Warn(s.logger).Log(err)
		return err
	}
	for {
		select {
		case <-ctx.Done():

			return nil
		default:
			if loginSid == nil {
				err := s.Login()
				if err != nil {
					level.Warn(s.logger).Log("Failed to login")
					return err
				}
			}
			err := s.Scrape()
			if err != nil {
				level.Warn(s.logger).Log("Failed to scrape")
			}
			time.Sleep(15 * time.Second)
		}
	}
}

func (s *Scraper) Login() error {
	level.Debug(s.logger).Log("logging in")
	uri := "login_sid.lua"
	url := s.cfg.FritzBoxURL + uri
	resp, err := http.Get(url)
	if err != nil {
		level.Warn(s.logger).Log("Error logging in", err)
		return err
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		// expected: <?xml version=\"1.0\" encoding=\"utf-8\"?><SessionInfo><SID>0000000000000000</SID><Challenge>abababababa</Challenge><BlockTime>0</BlockTime><Rights></Rights></SessionInfo>
		c := &fritz.LoginChallenge{}
		err := xml.Unmarshal([]byte(body), &c)
		if err != nil {
			level.Warn(s.logger).Log("Error parsing xml", err)
			return err
		}

		if c.SID == "0000000000000000" {
			//cpstr := strings.ToLower(c.Challenge + "-" + s.cfg.Password)
			cpstr := c.Challenge + "-" + s.cfg.Password
			md5sum := GetMD5Hash(cpstr)
			responseCode := c.Challenge + "-" + md5sum
			resp, err := http.Get(url + "?user=" + s.cfg.Username + "&response=" + responseCode)
			if err != nil {
				level.Warn(s.logger).Log("Error on presenting response", err)
				return err
			} else {
				defer resp.Body.Close()
				body, _ := ioutil.ReadAll(resp.Body)
				// expected: <?xml version=\"1.0\" encoding=\"utf-8\"?><SessionInfo><SID>0000000000000000</SID><Challenge>abababababa</Challenge><BlockTime>0</BlockTime><Rights></Rights></SessionInfo>
				level.Debug(s.logger).Log("Response", body)
				err := xml.Unmarshal([]byte(body), &c)
				if err != nil {
					level.Warn(s.logger).Log("Error parsing xml", err)
					return err
				}
				if c.SID == "0000000000000000" {
					level.Warn(s.logger).Log("Credentials invalid")
					return errors.New("Login failed")
				} else {
					loginSid = &c.SID
				}
			}
		}
		time.Sleep(time.Duration(c.BlockTime) * time.Second)
	}
	return nil
}

func GetMD5Hash(text string) string {
	hasher := md5.New()
	codes := utf16.Encode([]rune(text))
	b := make([]byte, len(codes)*2)
	for i, r := range codes {
		b[i*2] = byte(r)
		b[i*2+1] = byte(r >> 8)
	}
	_, err := hasher.Write([]byte(b))
	if err != nil {
		print("error")
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *Scraper) Scrape() error {
	level.Debug(s.logger).Log("lastLogTime", lastLogTime.Format(time.RFC3339))

	landevices, _ := s.query("query.lua", "network=landevice:settings/landevice/list(name,ip,mac,UID,dhcp,wlan,ethernet,active,wakeup,deleteable,source,online,speed,guest,url,devtype)", "GET", nil)
	level.Debug(s.logger).Log(landevices)
	l := &fritz.LanDevices{}
	err := l.Decode(landevices)
	if err != nil {
		// usually login expired
		err := s.Login()
		if err != nil {
			level.Warn(s.logger).Log("Error", err)
		}
		landevices, _ := s.query("query.lua", "network=landevice:settings/landevice/list(name,ip,mac,UID,dhcp,wlan,ethernet,active,wakeup,deleteable,source,online,speed,guest,url,devtype)", "GET", nil)
		err = l.Decode(landevices)
		if err != nil {
			level.Warn(s.logger).Log("Error", err)
		}
	} else {
		for _, v := range l.Network {
			// get specific infos
			devType := "N/A"
			active, _ := strconv.ParseFloat(v.Active, 64)
			online, _ := strconv.ParseFloat(v.Online, 64)
			speed, _ := strconv.ParseFloat(v.Speed, 64)
			if online == 1 {
				fd, err := s.deviceSpecificData(v.UID)
				if err == nil {
					devType = fd.DevType
					if fd.DevType == "wlan" {
						labels := prometheus.Labels{
							"name":     v.Name,
							"ip":       v.IP,
							"mac":      v.Mac,
							"dev_type": fd.DevType,
						}
						WlanDeviceSignal.With(labels).Set(fd.Wlan.Rssi)
						labels["direction"] = "tx"
						WlanDeviceSpeed.With(labels).Set(fd.Wlan.Speed)
						WlanDeviceSpeedMax.With(labels).Set(fd.Wlan.SpeedTxMax)
						labels["direction"] = "rx"
						WlanDeviceSpeed.With(labels).Set(fd.Wlan.SpeedRx)
						WlanDeviceSpeedMax.With(labels).Set(fd.Wlan.SpeedRxMax)

						delete(labels, "direction")
						labels["standard"] = fd.Wlan.WlanStandard
						labels["band"] = fd.Wlan.Band
						labels["encryption"] = fd.Wlan.Encryption
						oldLabels := s.deviceband[v.Name]
						WlanDeviceInfo.Delete(oldLabels)
						WlanDeviceInfo.With(labels).Set(1)
						s.deviceband[v.Name] = labels
					}
				}
			}
			LanDevicesActive.WithLabelValues(v.Name, v.IP, v.Mac, devType).Set(active)
			LanDevicesOnline.WithLabelValues(v.Name, v.IP, v.Mac, devType).Set(online)
			LanDevicesSpeed.WithLabelValues(v.Name, v.IP, v.Mac, devType).Set(speed)
		}
	}

	// Traffic Monitor is changed quite a lot since 7.57, haven't figured out yet, how to best get the data out of it, see Result below

	//old
	//trafficmon, _ := s.query("internet/inetstat_monitor.lua", "action=get_graphic&myXhr=1&xhr=1&useajay=1", "GET", nil)
	//new
	//	tmd := url.Values{}
	//	tmd.Set("page", "netMoni")
	//	tmd.Set("xhrId", "updateGraphs")
	//	tmd.Set("useajax", "1")
	//	trafficmon, _ := s.query("data.lua", "lang=de&xhr=1", "POST", tmd)
	//	// Result:
	//	// {"pid":"netMoni","hide":{"shareUsb":true,"liveTv":true,"dectRdio":true,"rrd":true,"rss":true,"ssoSet":true,"dectMail":true,"mobile":true,"liveImg":true},"timeTillLogout":"1200","time":[],"data":{"show_guest":true,"sync_groups":[{"us_bps_curr_max":21983,"us_default_bps_curr":[14488,3593,7839,21655,2771,8319,15221,2849,10985,17718,15000,9030,12844,4512,8269,16119,2282,7630,12540,2963],"ds_bps_max":29492068,"_node":"sg0","mode":"CABLE","ds_mc_bps_curr":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"ds_bps_curr":[107073,4251,3578,462893,2855,4297,104583,2897,331200,8155,346743,4855,3908,4212,4981,13479,3152,3983,3522,2952],"us_bps_max":539444,"dynamic":true,"us_realtime_bps_curr":[292,321,292,328,292,291,292,319,291,352,409,329,363,291,291,386,292,292,320,291],"downstream":3600000,"upstream":384000,"name":"sync_cable","guest_us_bps":[0],"ds_guest_bps_curr":[0,0,0,44,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"us_background_bps_curr":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],"ds_bps_curr_max":462893,"us_important_bps_curr":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}]},"sid":"..."}
	//	level.Info(s.logger).Log("trafficmon", trafficmon)
	//	t, err := fritz.DecodeTrafficMonitoringData(trafficmon)
	//	if err != nil {
	//		level.Warn(s.logger).Log("Error", err)
	//	} else {
	//		InternetDownstreamSpeed.WithLabelValues("internet").Set(t.DownstreamInternet[1])
	//		InternetDownstreamSpeed.WithLabelValues("media").Set(t.DownstreamMedia[1])
	//		if len(t.DownstreamGuest) > 0 {
	//			InternetDownstreamSpeed.WithLabelValues("guest").Set(t.DownstreamGuest[1])
	//		}
	//		InternetUpstreamSpeed.WithLabelValues("realtime").Set(t.UpstreamRealtime[1])
	//		InternetUpstreamSpeed.WithLabelValues("high").Set(t.UpstreamHighPriority[1])
	//		InternetUpstreamSpeed.WithLabelValues("default").Set(t.UpstreamDefaultPriority[1])
	//		InternetUpstreamSpeed.WithLabelValues("low").Set(t.UpstreamLowPriority[1])
	//		if len(t.UpstreamGuest) > 0 {
	//			InternetDownstreamSpeed.WithLabelValues("guest").Set(t.UpstreamGuest[1])
	//		}
	//
	//		level.Debug(s.logger).Log("traffic", t.Mode, "downstream", t.DownstreamCurrentMax, "upstream", t.UpstreamCurrentMax)
	//
	//	}

	out, err := s.query("internet/inetstat_counter.lua", "csv=", "GET", nil)
	if err != nil {
		//ignore for now
	} else {
		level.Debug(s.logger).Log("inetstat-counter", out)
	}
	//s.connectionInfos = &newLabels
	//	ConnectionInfo.With(newLabels).Set(1)
	//out, err := s.query("internet/inetstat_counter.lua", "csv=", "GET", nil)
	//if err != nil {
	//	//ignore for now
	//} else {
	//	level.Info(s.logger).Log("inetstat-counter", out)
	//}

	//s.generalInfo()
	//systemstatus, _ := s.query("cgi-bin/system_status","")
	//wlan, _ := s.query("data.lua","xhr=1&xhrId=wlanDevices&useajax=1&no_siderenew=&lang=de")

	if s.cfg.LogPath != "" {
		s.queryLogs()
	}

	return nil
}

func (s *Scraper) loadServices() error {
	device := regexp.MustCompile("http.*://(.*)/.*").FindAllStringSubmatch(s.cfg.FritzBoxURL, 1)
	root, err := fritzbox_upnp.LoadServices(device[0][1], 49000)
	if err != nil {
		return err
	}
	s.upnpServicesRoot = root
	for name, se := range root.Services {
		for a, _ := range se.Actions {

			level.Debug(s.logger).Log("Services, Action", name, a)
		}
	}
	return err
}

func (s *Scraper) getLinkInfo() (bytesSent, bytesReceived float64, wanAccessType, linkStatus string, upstream, downstream float64) {
	if s.upnpServicesRoot == nil {
		level.Warn(s.logger).Log("Services not loaded")
		return
	}
	service, _ := s.upnpServicesRoot.Services["urn:schemas-upnp-org:service:WANCommonInterfaceConfig:1"]

	action, _ := service.Actions["GetAddonInfos"]
	res, err := action.Call()
	if err != nil {
		level.Warn(s.logger).Log("Failed to execute action call", "action", action.Name)
	}
	resbytesSent, _ := res["TotalBytesSent"]
	switch v := resbytesSent.(type) {
	case uint64:
		bytesSent = float64(v)
	}
	resBytesReceived, _ := res["TotalBytesReceived"]
	switch v := resBytesReceived.(type) {
	case uint64:
		bytesReceived = float64(v)
	}

	action, _ = service.Actions["GetCommonLinkProperties"]
	res, err = action.Call()
	if err != nil {
		level.Warn(s.logger).Log("Failed to execute action call", action.Name)
	}
	resWanAccessType, _ := res["WANAccessType"]
	switch v := resWanAccessType.(type) {
	case string:
		wanAccessType = v
	}
	resLinkStatus, _ := res["PhysicalLinkStatus"]
	switch v := resLinkStatus.(type) {
	case string:
		linkStatus = v
	}
	resUpstream, _ := res["Layer1UpstreamMaxBitRate"]
	switch tval := resUpstream.(type) {
	case uint64:
		upstream = float64(tval)

	}
	resDownstream, _ := res["Layer1DownstreamMaxBitRate"]
	switch tval := resDownstream.(type) {
	case uint64:
		downstream = float64(tval)

	}

	return
}

func (s *Scraper) getConnectionInfo() (extIPV6, extIPV4, connectionStatus, connectionError string, uptime float64) {
	if s.upnpServicesRoot == nil {
		level.Warn(s.logger).Log("Services not loaded")
		return
	}
	service, ok := s.upnpServicesRoot.Services["urn:schemas-upnp-org:service:WANIPConnection:1"]
	if !ok {
		level.Warn(s.logger).Log("Service not available")
		return
	}
	action, ok := service.Actions["X_AVM_DE_GetExternalIPv6Address"]
	if !ok {
		level.Warn(s.logger).Log("Action X_AVM_DE_GetExternalIPv6Address not available")
		return
	}
	res, err := action.Call()
	if err != nil {
		level.Warn(s.logger).Log("Failed to execute action call", action.Name)
	}
	resExtIpV6, _ := res["ExternalIPv6Address"]
	switch v := resExtIpV6.(type) {
	case string:
		extIPV6 = string(v)
	}

	action, _ = service.Actions["GetExternalIPAddress"]
	res, err = action.Call()
	if err != nil {
		level.Warn(s.logger).Log("Failed to execute action call", action.Name)
	}
	resExtIpV4, _ := res["ExternalIPAddress"]
	switch v := resExtIpV4.(type) {
	case string:
		extIPV4 = string(v)
	}

	action, _ = service.Actions["GetStatusInfo"]
	res, err = action.Call()
	if err != nil {
		level.Warn(s.logger).Log("Failed to execute action call", action.Name)
	}
	resConnStatus, _ := res["ConnectionStatus"]
	switch v := resConnStatus.(type) {
	case string:
		connectionStatus = string(v)
	}
	resConnError, _ := res["LastConnectionError"]
	switch v := resConnError.(type) {
	case string:
		connectionError = string(v)
	}
	resUptime, _ := res["Uptime"]
	switch tval := resUptime.(type) {
	case uint64:
		uptime = float64(tval)

	}

	return
}

func (s *Scraper) deviceSpecificData(UID string) (fritz.NetDevice, error) {
	dData := url.Values{}
	dData.Set("xhr", "1")
	dData.Set("xhrId", "all")
	dData.Set("lang", "de")
	dData.Set("dev", UID)
	dData.Set("page", "edit_device")
	dData.Set("initialRefreshParamsSaved", "true")
	dData.Set("no_siderenew", "")

	var fd fritz.NetDevice

	devData, err := s.query("data.lua", "", "POST", dData)
	if err != nil {
		level.Warn(s.logger).Log("error", err)
	} else {
		//level.Debug(s.logger).Log("UID",UID,"data",devData)
		fd, err = fritz.DecodeSingleDevice(devData)
		if err != nil {
			level.Warn(s.logger).Log("message", "Decoding failed", "error", err)
		}
	}
	return fd, err
}

//
//func (s *Scraper) generalInfo() {
//	dData := url.Values{}
//	dData.Set("xhr", "1")
//	dData.Set("xhrId", "all")
//	dData.Set("lang", "de")
//	dData.Set("page", "overview")
//	//dData.Set("initialRefreshParamsSaved", "true")
//	dData.Set("no_siderenew", "")
//
//	overviewData, err := s.query("data.lua", "", "POST", dData)
//	if err != nil {
//		level.Warn(s.logger).Log("Failure querying overview data", "error", err)
//	} else {
//		level.Debug(s.logger).Log("data", overviewData)
//
//		overview, err := fritz.DecodeOverViewData(overviewData)
//		if err != nil {
//			level.Warn(s.logger).Log("Failure parsing overview data", "error", err)
//		} else {
//			level.Info(s.logger).Log("FritzBox", overview.FritzOS.ProductName, "FritzOS", overview.FritzOS.Version)
//			level.Info(s.logger).Log("uptime", overview.Internet.Uptime, "internet", overview.Internet.Txt)
//		}
//	}
//}

func (s *Scraper) queryLogs() {
	logData := url.Values{}
	logData.Set("page", "log")
	logData.Set("xhr", "1")
	logData.Set("xhrId", "all")
	logData.Set("lang", "de")
	logData.Set("no_siderenew", "")
	logData.Set("filter", "0")

	logs, err := s.query("data.lua", "", "POST", logData)
	if err != nil {
		level.Warn(s.logger).Log("Error", err)
	}
	level.Debug(s.logger).Log("logs", logs)
	loglines := &fritz.Logs{}
	err = loglines.Decode(logs)
	if err != nil {
		level.Warn(s.logger).Log("Error", err)
	} else {
		// process log lines -> write to file on disk?
		jsonLines, _ := loglines.EncodeAfter(*lastLogTime)
		err = s.logPusher.Push(jsonLines)
		if err != nil {
			level.Warn(s.logger).Log("message", "cannot send logs", "error", err)
		}
		f, err := os.OpenFile(s.cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			level.Warn(s.logger).Log("Failed to open fritz-log-path", s.cfg.LogPath, "err", err)
		} else {
			defer f.Close()
			for _, line := range jsonLines {
				_, err := f.WriteString(string(line) + "\n")
				if err != nil {
					level.Warn(s.logger).Log("message", "Failed to write to file", "err", err)
				}
			}
			err := f.Sync()
			if err != nil {
				level.Warn(s.logger).Log("message", "Failed to sync buffer with file", "err", err)
			}
		}
		*lastLogTime = loglines.Data.LogLines[0].Timestamp
	}

}

func (s *Scraper) query(path string, options string, method string, urlData url.Values) (string, error) {
	if options == "" {
		options = "0=0"
	}
	var body io.Reader
	var uri string
	if urlData != nil {
		urlData.Set("sid", *loginSid)
		body = strings.NewReader(urlData.Encode())
		uri = s.cfg.FritzBoxURL + "/" + path
	} else {
		uri = s.cfg.FritzBoxURL + "/" + path + "?sid=" + *loginSid + "&" + options

	}

	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	request, err := http.NewRequest(method, uri, body)
	if err != nil {
		level.Warn(s.logger).Log("Invalid request", err)
	}
	request.Header.Set("Content-type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)

	if err != nil {
		level.Warn(s.logger).Log("Error logging in", err)
		return "", err
	} else {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		//level.Debug(s.logger).Log("body", body)
		return string(body), nil
	}

}

func getOtherBand(band string) string {
	if band == "5 Ghz" {
		return "2,4 Ghz"
	} else {
		return "5 Ghz"
	}
}

func removeString(arr []string, name string) []string {
	if len(arr) == 0 {
		return arr
	}
	var it int

	for i, j := range arr {
		if j == name {
			it = i
		}
	}
	return append(arr[:it], arr[it+1:]...)
}
