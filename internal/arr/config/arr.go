package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"

	base_config "github.com/onedr0p/exportarr/internal/config"
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
	ApiKey                  string         `koanf:"-"` // stores the API key
	DisableSSLVerify        bool           `koanf:"-"` // stores the disable SSL verify flag
	Prowlarr                ProwlarrConfig `koanf:"prowlarr"`
	k                       *koanf.Koanf
}

func (c *ArrConfig) UseBasicAuth() bool {
	return !c.FormAuth && c.AuthUsername != "" && c.AuthPassword != ""
}

func (c *ArrConfig) UseFormAuth() bool {
	return c.FormAuth
}

// URLLabel() exists for backwards compatibility -- prior versions built the URL in the client,
// meaning that the "url" metric label was missing the Port & base path that the XMLConfig provided.
func (c *ArrConfig) URLLabel() string {
	if c.XMLConfig != "" {
		u, err := url.Parse(c.URL)
		if err != nil {
			// Should be unreachable as long as we validate that the URL is valid in LoadConfig/Validate
			return "Could Not Parse URL"
		}
		return u.Scheme + "://" + u.Host
	}
	return c.URL
}

func arrMergeFunc(src, dest map[string]interface{}) error {
	dest["arr"] = src
	return nil
}

func LoadArrConfig(conf base_config.Config, flags *flag.FlagSet) (*ArrConfig, error) {
	k := koanf.New(".")
	if err := k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		return nil, err
	}

	// XMLConfig
	xmlConfig := k.String("config")
	if xmlConfig != "" {
		err := k.Load(file.Provider(xmlConfig), XMLParser(), koanf.WithMergeFunc(XMLParser().Merge))
		if err != nil {
			return nil, err
		}
	}

	err := k.Load(env.Provider("", ".", func(s string) string {
		s = strings.ToLower(s)
		s = strings.Replace(s, "__", ".", -1)
		s = strings.Replace(s, "_", "-", -1)
		return backwardsCompatibilityTransforms(s)
	}), nil, koanf.WithMergeFunc(arrMergeFunc))
	if err != nil {
		return nil, err
	}

	out := &ArrConfig{
		URL:              conf.URL,
		ApiKey:           conf.ApiKey,
		DisableSSLVerify: conf.DisableSSLVerify,
		k:                k,
	}

	if err = k.Unmarshal("arr", out); err != nil {
		return nil, err
	}

	u, err := url.JoinPath(out.URL, "api", out.ApiVersion)
	if err != nil {
		return nil, err
	}
	out.URL = u
	return out, nil
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

// Remove in v2.0.0
func backwardsCompatibilityNormalizeFunc(f *flag.FlagSet, name string) flag.NormalizedName {
	if name == "basic-auth-username" {
		return flag.NormalizedName("auth-username")
	}
	if name == "basic-auth-password" {
		return flag.NormalizedName("auth-password")
	}
	return flag.NormalizedName(name)
}

// Remove in v2.0.0
func backwardsCompatibilityTransforms(s string) string {
	switch s {
	case "basic-auth-username":
		return "auth-username"
	case "basic-auth-password":
		return "auth-password"
	default:
		return s
	}
}
