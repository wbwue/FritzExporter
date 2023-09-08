package loki

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type Pusher struct {
	URL    string
	client *resty.Client
}

func New(lokiURL string) Pusher {
	pusher := Pusher{}
	pusher.client = resty.New()
	pusher.URL = lokiURL + "loki/api/v1/push"

	return pusher
}

func (p *Pusher) Push(lines [][]byte) error {
	if len(lines) == 0 {
		return nil
	}
	tmpbody := "{\"streams\": [{ \"stream\": { \"app\": \"fritzbox\" }, \"values\": ["
	now := fmt.Sprint(time.Now().UnixNano())
	for _, l := range lines {
		tmpbody += "[\"" + now + "\"," + strconv.Quote(string(l)) + "]," // 20060102150405
		//tmpbody += "[\""+now+"\",\""+strings.ReplaceAll(string(l),"\"","\\\"")+"\"]," // 20060102150405
	}
	body := tmpbody[:len(tmpbody)-1] + "] }]}"
	//	fmt.Println("message: log push result, request: "+ body)
	resp, err := p.client.R().SetHeader("Content-Type", "application/json").SetBody(body).Post(p.URL)
	if err != nil {
		return err
	}
	fmt.Println("message: log push result, response: " + resp.String())
	return nil
}
