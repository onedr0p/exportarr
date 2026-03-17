package config

import (
	"errors"
	"net/url"

	"gopkg.in/yaml.v3"
)

type yamlConfig struct {
	Auth struct {
		APIKey string `yaml:"apikey"`
	} `yaml:"auth"`
}

type YAML struct{}

func YAMLParser() *YAML {
	return &YAML{}
}

func (p *YAML) Unmarshal(b []byte) (map[string]interface{}, error) {
	var config yamlConfig
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	ret := map[string]interface{}{
		"api-key": config.Auth.APIKey,
	}
	return ret, nil
}

func (p *YAML) Marshal(o map[string]interface{}) ([]byte, error) {
	return nil, errors.New("not implemented")
}

func (p *YAML) Merge(baseURL string) func(src, dest map[string]interface{}) error {
	return func(src, dest map[string]interface{}) error {
		if src["api-key"] != nil && src["api-key"].(string) != "" {
			dest["api-key"] = src["api-key"]
		}

		u, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		dest["url"] = u.String()
		return nil
	}
}
