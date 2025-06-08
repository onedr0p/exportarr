package config

import (
	"github.com/gookit/validate"
	base_config "github.com/shamelin/exportarr/internal/config"
)

type SabnzbdConfig struct {
	URL              string `validate:"required|url"`
	ApiKey           string `validate:"required"`
	AuthUsername     string
	AuthPassword     string
	DisableSSLVerify bool
}

func LoadSabnzbdConfig(conf base_config.Config) (*SabnzbdConfig, error) {
	ret := &SabnzbdConfig{
		URL:              conf.URL,
		ApiKey:           conf.ApiKey,
		AuthUsername:     conf.AuthUsername,
		AuthPassword:     conf.AuthPassword,
		DisableSSLVerify: conf.DisableSSLVerify,
	}
	return ret, nil
}

func (c *SabnzbdConfig) Validate() error {
	v := validate.Struct(c)
	if !v.Validate() {
		return v.Errors
	}
	return nil
}
