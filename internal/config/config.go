// Package config loads and validates exportarr's base configuration from
// environment variables and flags.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	flag "github.com/spf13/pflag"
)

// RegisterConfigFlags registers the base exportarr flags on the given FlagSet.
func RegisterConfigFlags(flags *flag.FlagSet) {
	flags.StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	flags.String("log-format", "console", "Log format (console, json)")
	flags.StringP("url", "u", "", "URL to *arr instance")
	flags.StringP("api-key", "a", "", "API Key for *arr instance")
	flags.Bool("disable-ssl-verify", false, "Disable SSL verification")
	flags.StringP("interface", "i", "", "IP address to listen on")
	flags.IntP("port", "p", 0, "Port to listen on")
	flags.Duration("request-timeout", 0, "HTTP timeout per request to the target app")
}

// Config is the base configuration shared by every exportarr subcommand.
type Config struct {
	App       string `env:"-"`
	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"console"`
	URL       string `env:"URL"`
	// Secret-bearing variables carry the `unset` option: the env library
	// removes them from the process environment after parsing, so they are
	// not visible in /proc/<pid>/environ or inherited by child processes.
	APIKey string `env:"API_KEY,unset"`
	// APIKeyFromFile receives the *contents* of the file named by API_KEY_FILE
	// (the env library's `file` option) — Docker/Kubernetes secrets mounts.
	APIKeyFromFile   string        `env:"API_KEY_FILE,file,unset"`
	Port             int           `env:"PORT" envDefault:"8081"`
	Interface        string        `env:"INTERFACE" envDefault:"0.0.0.0"`
	DisableSSLVerify bool          `env:"DISABLE_SSL_VERIFY"`
	RequestTimeout   time.Duration `env:"REQUEST_TIMEOUT" envDefault:"60s"`
}

// OverlayFlag copies the value of an explicitly-set flag into dst, so flags
// win over environment-derived configuration. The getter is one of the typed
// FlagSet accessors (GetString, GetInt, GetBool, ...).
func OverlayFlag[T any](flags *flag.FlagSet, name string, get func(string) (T, error), dst *T) {
	if !flags.Changed(name) {
		return
	}
	if v, err := get(name); err == nil {
		*dst = v
	}
}

// LoadConfig parses environment variables into a Config, then overlays any
// explicitly-set flags (flags win over environment).
func LoadConfig(flags *flag.FlagSet) (*Config, error) {
	out := &Config{}
	if err := env.Parse(out); err != nil {
		return nil, err
	}

	OverlayFlag(flags, "log-level", flags.GetString, &out.LogLevel)
	OverlayFlag(flags, "log-format", flags.GetString, &out.LogFormat)
	OverlayFlag(flags, "url", flags.GetString, &out.URL)
	OverlayFlag(flags, "api-key", flags.GetString, &out.APIKey)
	OverlayFlag(flags, "interface", flags.GetString, &out.Interface)
	OverlayFlag(flags, "port", flags.GetInt, &out.Port)
	OverlayFlag(flags, "disable-ssl-verify", flags.GetBool, &out.DisableSSLVerify)
	OverlayFlag(flags, "request-timeout", flags.GetDuration, &out.RequestTimeout)

	// A mounted secret wins over any inline API_KEY. Secrets commonly end
	// with a newline, which the env library preserves: trim it.
	if out.APIKeyFromFile != "" {
		out.APIKey = strings.TrimSpace(out.APIKeyFromFile)
	}
	return out, nil
}

// Validate checks the configuration against its validation rules.
func (c *Config) Validate() error {
	var errs []error
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(c.LogLevel)); err != nil {
		errs = append(errs, errors.New("log-level must be one of: debug, info, warn, error"))
	}
	if c.LogFormat != "console" && c.LogFormat != "json" {
		errs = append(errs, errors.New("log-format must be one of: console, json"))
	}
	if c.Port == 0 {
		errs = append(errs, errors.New("port is required"))
	}
	if net.ParseIP(c.Interface) == nil {
		errs = append(errs, fmt.Errorf("interface must be a valid IP address: %q", c.Interface))
	}
	return errors.Join(errs...)
}
