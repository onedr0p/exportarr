// Package config holds SABnzbd-specific exportarr configuration.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	base_config "github.com/onedr0p/exportarr/internal/config"
)

// SabnzbdConfig is the configuration for the SABnzbd exporter.
type SabnzbdConfig struct {
	URL              string
	APIKey           string
	DisableSSLVerify bool
	RequestTimeout   time.Duration
}

// LoadSabnzbdConfig builds a SabnzbdConfig from the base configuration.
func LoadSabnzbdConfig(conf base_config.Config) (*SabnzbdConfig, error) {
	ret := &SabnzbdConfig{
		URL:              conf.URL,
		APIKey:           conf.APIKey,
		DisableSSLVerify: conf.DisableSSLVerify,
		RequestTimeout:   conf.RequestTimeout,
	}
	return ret, nil
}

// Validate checks the configuration against its validation rules.
func (c *SabnzbdConfig) Validate() error {
	var errs []error
	if c.URL == "" {
		errs = append(errs, errors.New("url is required"))
	} else if u, err := url.Parse(c.URL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("url must be a valid URL: %q", c.URL))
	}
	if c.APIKey == "" {
		errs = append(errs, errors.New("api-key is required"))
	}
	return errors.Join(errs...)
}
