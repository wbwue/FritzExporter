package fritz

import (
	"github.com/Jeffail/gabs/v2"
)

type Overview struct {
	FritzOS  FritzOS
	Internet Internet
}

type FritzOS struct {
	ProductName string
	Version     string
}

type Internet struct {
	Txt    string
	Uptime float64
}

func DecodeOverViewData(body string) (Overview, error) {
	var ov Overview
	jsonParsed, err := gabs.ParseJSON([]byte(body))
	if err != nil {
		return ov, err
	} else {
		fos := FritzOS{}
		fritzos := jsonParsed.Path("data.fritzos")
		fos.ProductName = fritzos.Path("Productname").Data().(string)
		fos.Version = fritzos.Path("nspver").Data().(string)
		ov.FritzOS = fos
		internet := Internet{}
		inet := jsonParsed.Path("data.internet")
		internet.Txt = inet.Path("txt").Children()[0].Data().(string)
		ov.Internet = internet
	}
	return ov, nil
}
