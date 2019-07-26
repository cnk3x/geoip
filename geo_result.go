package geoip

import (
	"encoding/xml"
	"strings"
)

//GeoResult GeoResult
type GeoResult struct {
	XMLName   xml.Name `json:"-" xml:"result"`
	IP        string   `json:"ip,omitempty" xml:"ip,attr,omitempty"`
	Lang      string   `json:"lang,omitempty" xml:"lang,attr,omitempty"`
	Code      string   `json:"code,omitempty" xml:"code,attr,omitempty"`
	Msg       string   `json:"msg,omitempty" xml:"msg,omitempty"`
	Continent *Name    `json:"continent,omitempty" xml:"continent,omitempty"`
	Country   *Name    `json:"country,omitempty" xml:"country,omitempty"`
	City      *Name    `json:"city,omitempty" xml:"city,omitempty"`
}

func (res *GeoResult) String() string {
	txts := make([]string, 0)
	if res.City != nil {
		txts = append(txts, res.City.Name)
	}
	if res.Country != nil {
		txts = append(txts, res.Country.Name)
	}
	if res.Continent != nil {
		txts = append(txts, res.Continent.Name)
	}
	txts = append(txts, res.IP)
	return strings.Join(txts, ",")
}

//Name Name
type Name struct {
	Code string `json:"code,omitempty" xml:"code,attr,omitempty"`
	Name string `json:"name,omitempty" xml:",innerxml"`
}

var langs = []string{"en", "zh-CN", "de", "es", "fr", "ja", "pt-BR", "ru"}

//SetName SetName
func (n *Name) SetName(lang string, langMap map[string]string) {
	if name, find := langMap[lang]; find {
		n.Name = name
	} else {
		for _, lang := range langs {
			if name, find := langMap[lang]; find && name != "" {
				n.Name = name
				return
			}
		}
		for _, name := range langMap {
			n.Name = name
		}
	}
}
