package fritz

import (
	"github.com/Jeffail/gabs/v2"
	"strconv"
)

type NetDevice struct {
	DevType  string
	Mac      string
	State    string
	Name     string
	Wlan     NetWlan
	Ethernet NetEthernet
	Speed    int64
}

type NetEthernet struct {
	Port  string
	Speed int
}

type NetWlan struct {
	Mac          string  `json:"mac"`
	Flags        string  `json:"flags"`
	IsRepeater   string  `json:"is_repeater"`
	WmmActive    string  `json:"wmm_active"`
	Cipher       string  `json:"cipher"`
	Rssi         float64 `json:"rssi"`
	MuMimoGroup  bool    `json:"mu_mimo_group"`
	Streams      string  `json:"streams"`
	SpeedTxMax   float64 `json:"speed_tx_max"`
	SpeedRxMax   float64 `json:"speed_rx_max"`
	SpeedRx      float64 `json:"speed_rx"`
	ChannelWidth float64 `json:"channel_width"`
	Speed        float64 `json:"speed"`
	Mode         string  `json:"mode"`
	State        string  `json:"state"`
	Quality      float64 `json:"quality"`

	Band             string   `json:"band"`
	Signalstrength   float64  `json:"signalstrength"`
	WlanStandard     string   `json:"wlanStandard"`
	QualityOfService string   `json:"qualityOfService"`
	Encryption       string   `json:"encryption"`
	SignalProperties []string `json:"signalProperties"`
	CurRate          string   `json:"curRate"`
	MaxRate          string   `json:"maxRate"`
	Is5GHz           bool     `json:"is5GHz"`
}

func DecodeSingleDevice(body string) (NetDevice, error) {
	var fd NetDevice

	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		return fd, err
	}
	fd.DevType = jsonParsed.Path("data.vars.dev.devType").Data().(string)
	fd.Name = jsonParsed.Path("data.vars.dev.name.displayName").Data().(string)
	fd.State = jsonParsed.Path("data.vars.dev.state").Data().(string)
	if fd.DevType == "wlan" {
		show := jsonParsed.Path("data.vars.dev.wlan.show").Children()[0]

		fd.Wlan.Flags = show.Path("flags").Data().(string)
		fd.Wlan.IsRepeater = show.Path("is_repeater").Data().(string)
		fd.Wlan.WmmActive = show.Path("wmm_active").Data().(string)
		fd.Wlan.Cipher = show.Path("cipher").Data().(string)
		fd.Wlan.Rssi, _ = strconv.ParseFloat(show.Path("rssi").Data().(string), 64)
		fd.Wlan.MuMimoGroup = show.Path("mu_mimo_group").Data().(bool)
		fd.Wlan.Streams = show.Path("streams").Data().(string)
		fd.Wlan.SpeedTxMax, _ = strconv.ParseFloat(show.Path("speed_tx_max").Data().(string), 64)
		fd.Wlan.SpeedRxMax, _ = strconv.ParseFloat(show.Path("speed_rx_max").Data().(string), 64)
		fd.Wlan.SpeedRx, _ = strconv.ParseFloat(show.Path("speed_rx").Data().(string), 64)
		fd.Wlan.Speed, _ = strconv.ParseFloat(show.Path("speed").Data().(string), 64)
		fd.Wlan.ChannelWidth, _ = strconv.ParseFloat(show.Path("channel_width").Data().(string), 64)
		fd.Wlan.Mode = show.Path("mode").Data().(string)
		fd.Wlan.Mac = show.Path("mac").Data().(string)
		fd.Wlan.Quality, _ = strconv.ParseFloat(show.Path("quality").Data().(string), 64)

		dev := jsonParsed.Path("data.vars.dev.wlan.devs").Children()[0]
		fd.Wlan.Signalstrength, _ = strconv.ParseFloat(dev.Path("signalstrength").Data().(string), 64)
		fd.Wlan.Band = dev.Path("band").Data().(string)
		fd.Wlan.State = dev.Path("state").Data().(string)
		fd.Wlan.WlanStandard = dev.Path("wlanStandard").Data().(string)
		fd.Wlan.QualityOfService = dev.Path("qualityOfService").Data().(string)
		fd.Wlan.Encryption = dev.Path("encryption").Data().(string)
		fd.Wlan.SignalProperties = getStringArray(dev, "encryption")
		fd.Wlan.Is5GHz = dev.Path("is5GHz").Data().(bool)

	}
	if fd.DevType == "lan" {
		dev := jsonParsed.Path("data.vars.dev.topology.path.path").Children()[1]
		fd.Ethernet.Port = dev.Path("device.ethernetport").Data().(string)
		fd.Ethernet.Speed, _ = strconv.Atoi(dev.Path("device.speed").Data().(string))
	}

	return fd, nil
}

func getStringArray(json *gabs.Container, path string) []string {
	var vals []string
	for _, child := range json.Path(path).Children() {
		vals = append(vals, child.Data().(string))
	}
	return vals
}
