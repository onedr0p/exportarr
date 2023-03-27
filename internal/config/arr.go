package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

func RegisterArrFlags(flags *flag.FlagSet) {
	flags.StringP("interface", "i", "", "IP address to listen on")
	flags.Bool("disable-ssl-verify", false, "Disable SSL verification")
	flags.String("auth-username", "", "Username for basic or form auth")
	flags.String("auth-password", "", "Password for basic or form auth")
	flags.Bool("form-auth", false, "Use form based authentication")
	flags.Bool("enable-unknown-queue-items", false, "Enable unknown queue items")
	flags.Bool("enable-additional-metrics", false, "Enable additional metrics")

	// Backwards Compatibility - normalize function will hide these from --help. remove in v2.0.0
	flags.String("basic-auth-username", "", "Username for basic or form auth")
	flags.String("basic-auth-password", "", "Password for basic or form auth")
	flags.SetNormalizeFunc(backwardsCompatibilityNormalizeFunc)
}

type ArrConfig struct {
	App                     string         `koanf:"arr"`
	ApiVersion              string         `koanf:"api-version" validate:"required|in:v3,v4"`
	XMLConfig               string         `koanf:"config"`
	AuthUsername            string         `koanf:"auth-username"`
	AuthPassword            string         `koanf:"auth-password"`
	FormAuth                bool           `koanf:"form-auth"`
	EnableUnknownQueueItems bool           `koanf:"enable-unknown-queue-items"`
	EnableAdditionalMetrics bool           `koanf:"enable-additional-metrics"`
	URL                     string         `koanf:"-"` // stores rendered Arr URL (with api version)
	Prowlarr                ProwlarrConfig `koanf:"prowlarr"`
}

// URLLabel() exists for backwards compatibility -- prior versions built the URL in the client,
// meaning that the "url" metric label was missing the Port & base path that the XMLConfig provided.
func (c *Config) URLLabel() string {
	if c.Arr.XMLConfig != "" {
		u, err := url.Parse(c.URL)
		if err != nil {
			// Should be unreachable as long as we validate that the URL is valid in LoadConfig/Validate
			return "Could Not Parse URL"
		}
		return u.Scheme + "://" + u.Host
	}
	return c.URL
}

func (c *ArrConfig) UseBasicAuth() bool {
	return !c.FormAuth && c.AuthUsername != "" && c.AuthPassword != ""
}

func (c *ArrConfig) UseFormAuth() bool {
	return c.FormAuth
}

func arrMergeFunc(src, dest map[string]interface{}) error {
	dest["arr"] = src
	return nil
}

func (c *Config) LoadArrFlags(flags *flag.FlagSet) error {
	err := c.k.Load(posflag.Provider(flags, ".", c.k), nil, koanf.WithMergeFunc(arrMergeFunc))
	if err != nil {
		return err
	}

	err = c.k.Load(env.Provider("", ".", func(s string) string {
		s = strings.ToLower(s)
		s = strings.Replace(s, "__", ".", -1)
		s = strings.Replace(s, "_", "-", -1)
		return backwardsCompatibilityTransforms(s)
	}), nil, koanf.WithMergeFunc(arrMergeFunc))
	if err != nil {
		return err
	}

	if err := c.k.Unmarshal("arr", &c.Arr); err != nil {
		return err
	}

	u, err := url.JoinPath(c.URL, "api", c.Arr.ApiVersion)
	if err != nil {
		return err
	}
	c.Arr.URL = u
	return nil
}

func (c *ArrConfig) Validate() error {
	v := validate.Struct(c)
	if !v.Validate() {
		return v.Errors
	}

	if c.AuthPassword != "" && c.AuthUsername == "" {
		return fmt.Errorf("auth-username is required when auth-password is set")
	}
	if c.AuthUsername != "" && c.AuthPassword == "" {
		return fmt.Errorf("auth-password is required when auth-username is set")
	}
	if c.FormAuth && (c.AuthUsername == "" || c.AuthPassword == "") {
		return fmt.Errorf("auth-username and auth-password are required when form-auth is set")
	}

	return nil
}

func (c ArrConfig) Messages() map[string]string {
	return validate.MS{
		"ApiKey.regex":              "api-key must be a 32 character hex string",
		"LogLevel.ValidateLogLevel": "log-level must be one of: debug, info, warn, error, dpanic, panic, fatal",
	}
}

func (c ArrConfig) Translates() map[string]string {
	return validate.MS{
		"ApiVersion":              "api-version",
		"XMLConfig":               "config",
		"AuthUsername":            "auth-username",
		"AuthPassword":            "auth-password",
		"FormAuth":                "form-auth",
		"EnableUnknownQueueItems": "enable-unknown-queue-items",
		"EnableAdditionalMetrics": "enable-additional-metrics",
	}
}
