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

	ret := map[string]interface{}{
		"api-key":     config.ApiKey,
		"url-base":    config.UrlBase,
		"target-port": config.Port,
	}
	return ret, nil
}

func (p *XML) Marshal(o map[string]interface{}) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (p *XML) Merge(baseURL string) func(src, dest map[string]interface{}) error {
	return func(src, dest map[string]interface{}) error {
		if src["api-key"] != nil && src["api-key"].(string) != "" {
			dest["api-key"] = src["api-key"]
		}

		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}

		// Add or replace target port
		target_port := src["target-port"].(string)
		if len(target_port) > 0 {
			// AKA Port is defined in the config xml so we use this value as the source
			// of truth.
			u.Host = u.Hostname() + ":" + target_port

			// If Port is NOT defined in the XML it may be configured by the
			// *ARR__SERVER__PORT environment varible passed directly to *arr services.
			// In this case, we do not want to override whatever URL (that may include
			// the port) passed to exportarr by other means.
		}

		u = u.JoinPath(src["url-base"].(string))
		dest["url"] = u.String()
		return nil
	}
}
