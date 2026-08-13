// Package config holds SABnzbd-specific exportarr configuration.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"

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

// SabnzbdInstanceEnv is the per-instance configuration for SABnzbd in
// multi-app ("exportarr serve") mode, read from environment variables under
// a per-app prefix such as "SABNZBD__".
type SabnzbdInstanceEnv struct {
	URL              string        `env:"URL"`
	APIKey           string        `env:"API_KEY,unset"`
	APIKeyFromFile   string        `env:"API_KEY_FILE,file,unset"`
	DisableSSLVerify bool          `env:"DISABLE_SSL_VERIFY"`
	RequestTimeout   time.Duration `env:"REQUEST_TIMEOUT" envDefault:"60s"`
}

// Configured reports whether this instance has a URL set — the signal used
// to decide whether SABnzbd is enabled in multi-app mode.
func (i *SabnzbdInstanceEnv) Configured() bool {
	return i.URL != ""
}

// LoadSabnzbdInstanceEnv parses the per-app environment variables under
// prefix (e.g. "SABNZBD__") into a SabnzbdInstanceEnv. A mounted secret file
// (<PREFIX>API_KEY_FILE) wins over an inline <PREFIX>API_KEY, matching the
// base configuration's API_KEY_FILE behavior.
func LoadSabnzbdInstanceEnv(prefix string) (*SabnzbdInstanceEnv, error) {
	out := &SabnzbdInstanceEnv{}
	if err := env.ParseWithOptions(out, env.Options{Prefix: prefix}); err != nil {
		return nil, err
	}
	if out.APIKeyFromFile != "" {
		out.APIKey = strings.TrimSpace(out.APIKeyFromFile)
	}
	return out, nil
}

// NewSabnzbdConfigForInstance builds a SabnzbdConfig from one multi-app
// instance's environment.
func NewSabnzbdConfigForInstance(inst SabnzbdInstanceEnv) *SabnzbdConfig {
	return &SabnzbdConfig{
		URL:              inst.URL,
		APIKey:           inst.APIKey,
		DisableSSLVerify: inst.DisableSSLVerify,
		RequestTimeout:   inst.RequestTimeout,
	}
}
