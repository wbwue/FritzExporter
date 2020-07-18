package fritz

import (
	"encoding/json"
)

// TrafficMonitoringData holds the data for the up- and downstream traffic reported by the FRITZ!Box.
// codebeat:disable[TOO_MANY_IVARS]
type TrafficMonitoringData struct {
	Downstream      float64 `json:"downstream"`
	DownstreamMax     float64 `json:"ds_bps_max"`
	DownstreamCurrentMax     float64 `json:"ds_bps_curr_max"`
	DownstreamInternet      []float64 `json:"ds_bps_curr"`
	DownstreamMedia         []float64 `json:"ds_mc_bps_curr"`
	DownstreamGuest         []float64 `json:"ds_guest_bps_curr,omitempty"`
	Upstream      float64 `json:"upstream"`
	UpstreamMax      float64 `json:"us_bps_max"`
	UpstreamRealtime        []float64 `json:"us_realtime_bps_curr"`
	UpstreamCurrentMax     float64 `json:"us_bps_curr_max"`
	UpstreamHighPriority    []float64 `json:"us_important_bps_curr"`
	UpstreamDefaultPriority []float64 `json:"us_default_bps_curr"`
	UpstreamLowPriority     []float64 `json:"us_background_bps_curr"`
	UpstreamGuest           []float64 `json:"guest_us_bps,omitempty"`
	Mode 	string `json:"mode"`
}

// codebeat:enable[TOO_MANY_IVARS]

// BitsPerSecond returns a TrafficMonitoringData with metrics in units of bits/second.
func (d TrafficMonitoringData) BitsPerSecond() TrafficMonitoringData {
	return TrafficMonitoringData{
		DownstreamInternet:      d.DownstreamInternet,
		DownstreamMedia:         d.DownstreamMedia,
		DownstreamGuest:         d.DownstreamGuest,
		UpstreamRealtime:        d.UpstreamRealtime,
		UpstreamHighPriority:    d.UpstreamHighPriority,
		UpstreamDefaultPriority: d.UpstreamDefaultPriority,
		UpstreamLowPriority:     d.UpstreamLowPriority,
		UpstreamGuest:           d.UpstreamGuest,
	}
}

// KiloBitsPerSecond returns a TrafficMonitoringData with metrics in units of kbits/second.
func (d TrafficMonitoringData) KiloBitsPerSecond() TrafficMonitoringData {
	return TrafficMonitoringData{
		DownstreamInternet:      div(d.DownstreamInternet, 1024),
		DownstreamMedia:         div(d.DownstreamMedia, 1024),
		DownstreamGuest:         div(d.DownstreamGuest, 1024),
		UpstreamRealtime:        div(d.UpstreamRealtime, 1024),
		UpstreamHighPriority:    div(d.UpstreamHighPriority, 1024),
		UpstreamDefaultPriority: div(d.UpstreamDefaultPriority, 1024),
		UpstreamLowPriority:     div(d.UpstreamLowPriority, 1024),
		UpstreamGuest:           div(d.UpstreamGuest, 1024),
	}
}

func div(xs []float64, d float64) []float64 {
	ys := make([]float64, len(xs))
	for i, x := range xs {
		ys[i] = x / d
	}
	return ys
}


func DecodeTrafficMonitoringData(body string) (*TrafficMonitoringData,error) {
	t := &[]TrafficMonitoringData{}
	err := json.Unmarshal([]byte(body), t)

	if (err != nil) {
		return nil,err
	} else {
		for _,v := range *t {
			return &v,nil
		}
	}
	return nil, nil
}