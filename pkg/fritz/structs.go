package fritz

import (
	"encoding/xml"
)

type LoginChallenge struct {
	XMLName   xml.Name `xml:"SessionInfo"`
	SID       string   `xml:"SID"`
	Challenge string   `xml:"Challenge"`
	BlockTime int      `xml:BlockTime`
}
