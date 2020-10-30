package scraper

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/xml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"

	//	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
	"unicode/utf16"

	"github.com/wbwue/FritzExporter/pkg/config"
	"github.com/wbwue/FritzExporter/pkg/fritz"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

)

var (
	loginSid *string

	LanDevicesOnline = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_online",
		Help: "Gauge showing online state of device",
	},[]string{	"name", "ip", "mac"})
	LanDevicesActive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_active",
		Help: "Gauge showing active state of device",
	},[]string{	"name", "ip", "mac"})
	LanDevicesSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_lan_devices_speed",
		Help: "Gauge showing speed of device",
	},[]string{	"name", "ip", "mac"})

	InternetDownstreamSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_internet_downstream_current",
		Help: "Gauge showing latest internet downstream speed",
	},[]string{"type"})
	InternetUpstreamSpeed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "fritzbox_internet_upstream_current",
		Help: "Gauge showing latest internet upstream speed",
	},[]string{"type"})

)

type Scraper struct {
	cfg    *config.Config
	logger log.Logger
}

func NewScraper(config *config.Config, logger log.Logger) *Scraper {
	return &Scraper{
		cfg:    config,
		logger: logger,
	}
}

func (s *Scraper) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():

			return nil
		default:
			if loginSid == nil {
				err := s.Login()
				if (err != nil) {
					level.Warn(s.logger).Log("Failed to login")
				}
			}
			err := s.Scrape()
			if (err != nil) {
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
	if (err != nil) {
		print("error")
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func (s *Scraper) Scrape() error {

	landevices, _ := s.query("query.lua","network=landevice:settings/landevice/list(name,ip,mac,UID,dhcp,wlan,ethernet,active,wakeup,deleteable,source,online,speed,guest,url)")
	l := &fritz.LanDevices{}
	err := l.Decode(landevices)
	//err := json.Unmarshal([]byte(landevices), &l)
	if (err != nil) {
		// usually login expired
		err := s.Login()
		if (err != nil) {
			level.Warn(s.logger).Log("Error", err)
		}
		landevices, _ := s.query("query.lua","network=landevice:settings/landevice/list(name,ip,mac,UID,dhcp,wlan,ethernet,active,wakeup,deleteable,source,online,speed,guest,url)")
		err = l.Decode(landevices)
		if (err != nil) {
			level.Warn(s.logger).Log("Error", err)
		}
	} else {
		for _,v := range l.Network {
			active, _ := strconv.ParseFloat(v.Active, 64)
			online, _ := strconv.ParseFloat(v.Online, 64)
			speed, _ := strconv.ParseFloat(v.Speed, 64)
			LanDevicesActive.WithLabelValues(v.Name,v.IP,v.Mac).Set(active)
			LanDevicesOnline.WithLabelValues(v.Name,v.IP,v.Mac).Set(online)
			LanDevicesSpeed.WithLabelValues(v.Name,v.IP,v.Mac).Set(speed)
		}
	}
	trafficmon, _ := s.query("internet/inetstat_monitor.lua","action=get_graphic&myXhr=1&xhr=1&useajay=1")

	t, err := fritz.DecodeTrafficMonitoringData(trafficmon)
	if (err != nil) {
		level.Warn(s.logger).Log("Error", err)
	} else {
		InternetDownstreamSpeed.WithLabelValues("internet").Set(t.DownstreamInternet[1])
		InternetDownstreamSpeed.WithLabelValues("media").Set(t.DownstreamMedia[1])
		if len(t.DownstreamGuest) > 0 {
			InternetDownstreamSpeed.WithLabelValues("guest").Set(t.DownstreamGuest[1])
		}
		InternetUpstreamSpeed.WithLabelValues("realtime").Set(t.UpstreamRealtime[1])
		InternetUpstreamSpeed.WithLabelValues("high").Set(t.UpstreamHighPriority[1])
		InternetUpstreamSpeed.WithLabelValues("default").Set(t.UpstreamDefaultPriority[1])
		InternetUpstreamSpeed.WithLabelValues("low").Set(t.UpstreamLowPriority[1])
		if len(t.UpstreamGuest) > 0 {
			InternetDownstreamSpeed.WithLabelValues("guest").Set(t.UpstreamGuest[1])
		}

		level.Debug(s.logger).Log("traffic",t.Mode,"downstream",t.DownstreamCurrentMax,"upstream",t.UpstreamCurrentMax)

	}

	out, err := s.query("internet/inetstat_counter.lua","csv=")
	if (err != nil) {
		//ignore for now
	} else {
		level.Debug(s.logger).Log("inetstat-counter",out)
	}
	//systemstatus, _ := s.query("cgi-bin/system_status","")
	//wlan, _ := s.query("data.lua","xhr=1&xhrId=wlanDevices&useajax=1&no_siderenew=&lang=de")


	return nil
}

func (s *Scraper) query(path string, options string) (string, error) {
	if (options == "") {
		options = "0=0"
    }
	resp, err := http.Get(s.cfg.FritzBoxURL + "/" + path + "?sid=" + *loginSid + "&" + options)
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