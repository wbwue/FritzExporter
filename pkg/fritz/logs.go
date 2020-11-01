package fritz

import "encoding/json"

type Logs struct {
	Data Data `json:"data"`
}

type Data struct {
	Show Show `json:"show"`
	Filter string `json:"filter"`
	LogLines []LogLine
	LogFields [][]string `json:"log"`
	Timestamp int64 `json:"timestamp"`
}

type LogLine struct {
	Date string
	Time string
	Message string
	InfoCode string
	Filter string
	HelpUrl string
}

type Show struct {}

func (l *Logs) Decode(body string) error {
	err := json.Unmarshal([]byte(body), &l)
	for _,k := range l.Data.LogFields {
		line := LogLine{
			k[0],
			k[1],
			k[2],
			k[3],
			k[4],
			k[5],
		}
		l.Data.LogLines = append(l.Data.LogLines, line)
	}
	if (err != nil) {
		return err
	} else {
		return nil
	}
}
