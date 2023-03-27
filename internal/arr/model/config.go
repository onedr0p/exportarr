package model

import (
	"encoding/xml"
)

type Config struct {
	XMLName    xml.Name `xml:"Config"`
	ApiKey     string   `xml:"ApiKey"`
	Port       string   `xml:"Port"`
	UrlBase    string   `xml:"UrlBase"`
	ApiVersion string   `xml:"ApiVersion"`
}

func NewConfig() *Config {
	return &Config{
		ApiVersion: "v3",
	}
}
