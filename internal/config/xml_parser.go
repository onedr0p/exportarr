package config

import (
	"encoding/xml"
	"errors"
	"net/url"
)

type xmlConfig struct {
	XMLName xml.Name `xml:"Config"`
	ApiKey  string   `xml:"ApiKey"`
	Port    string   `xml:"Port"`
	UrlBase string   `xml:"UrlBase"`
}

type XML struct{}

func XMLParser() *XML {
	return &XML{}
}

func (p *XML) Unmarshal(b []byte) (map[string]interface{}, error) {
	var config xmlConfig
	if err := xml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"api-key":     config.ApiKey,
		"url-base":    config.UrlBase,
		"target-port": config.Port,
	}, nil
}

func (p *XML) Marshal(o map[string]interface{}) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (p *XML) Merge(src, dest map[string]interface{}) error {
	dest["api-key"] = src["api-key"]

	u, err := url.Parse(dest["url"].(string))
	if err != nil {
		return err
	}

	// Add or replace target port
	u.Host = u.Hostname() + ":" + src["target-port"].(string)
	u = u.JoinPath(src["url-base"].(string))
	dest["url"] = u.String()
	return nil
}
