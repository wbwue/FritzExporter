package fritz

import (
	"encoding/json"
	"time"
)

type Logs struct {
	Data Data `json:"data"`
}

type Data struct {
	Show      Show   `json:"show"`
	Filter    string `json:"filter"`
	LogLines  []LogLine
	LogFields [][]string `json:"log"`
	Timestamp int64      `json:"timestamp"`
}

type LogLine struct {
	Timestamp  time.Time `json:"timestamp"`
	Message    string    `json:"message"`
	InfoCode   string    `json:"info_code"`
	Filter     string    `json:"filter"`
	FilterName string    `json:"filter_name"`
	HelpUrl    string    `json:"help_url"`
}

func filterName(filterID string) string {
	switch filterID {
	case "1":
		return "System"
	case "2":
		return "Internetverbindung"
	case "3":
		return "Telefonie"
	case "4":
		return "WLAN"
	case "5":
		return "USB-Geräte"
	}
	return ""
}

// Filter values:
// 1: System, 2: Internetverbindung, 3: Telefonie, 4: WLAN, 5: USB-Geräte

type Show struct{}

func (l *Logs) Decode(body string) error {
	err := json.Unmarshal([]byte(body), &l)
	if err != nil {
		return err
	}
	for _, k := range l.Data.LogFields {
		date, err := time.ParseInLocation("02.01.06 15:04:05", k[0]+" "+k[1], time.Local)
		if err != nil {
			return err
		}
		line := LogLine{
			date,
			k[2],
			k[3],
			k[4],
			filterName(k[4]),
			k[5],
		}
		l.Data.LogLines = append(l.Data.LogLines, line)
	}
	return nil
}

func (l *Logs) Encode() (string, error) {
	resBytes, err := json.Marshal(l.Data.LogLines)
	if err != nil {
		return "", err
	} else {
		return string(resBytes), nil
	}
}

func (l *Logs) EncodeAfter(time time.Time) ([][]byte, error) {
	var latestLogs []LogLine
	latestLogs = l.Data.LogLines
	for k, v := range l.Data.LogLines {
		if v.Timestamp.Before(time) {
			if k > 0 {
				latestLogs = l.Data.LogLines[:k-1]
			} else {
				latestLogs = []LogLine{}
			}
			break
		}
	}
	jsonLines := [][]byte{}
	for _, line := range latestLogs {
		jsonLine, err := json.Marshal(line)
		if err != nil {
			return nil, err
		}
		jsonLines = append(jsonLines, jsonLine)
	}
	return jsonLines, nil

}
