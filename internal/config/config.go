package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

type Config struct {
	Arr                     string         `koanf:"arr"`
	LogLevel                string         `koanf:"log-level" validate:"ValidateLogLevel"`
	LogFormat               string         `koanf:"log-format" validate:"in:console,json"`
	URL                     string         `koanf:"url" validate:"required|url"`
	ApiKey                  string         `koanf:"api-key" validate:"required|regex:([a-z0-9]{32})"`
	ApiKeyFile              string         `koanf:"api-key-file"`
	ApiVersion              string         `koanf:"api-version" validate:"required|in:v3,v4"`
	XMLConfig               string         `koanf:"config"`
	Port                    int            `koanf:"port" validate:"required"`
	Interface               string         `koanf:"interface" validate:"required|ip"`
	DisableSSLVerify        bool           `koanf:"disable-ssl-verify"`
	AuthUsername            string         `koanf:"auth-username"`
	AuthPassword            string         `koanf:"auth-password"`
	FormAuth                bool           `koanf:"form-auth"`
	EnableUnknownQueueItems bool           `koanf:"enable-unknown-queue-items"`
	EnableAdditionalMetrics bool           `koanf:"enable-additional-metrics"`
	Prowlarr                ProwlarrConfig `koanf:"prowlarr"`
	k                       *koanf.Koanf
}

func (c *Config) UseBasicAuth() bool {
	return !c.FormAuth && c.AuthUsername != "" && c.AuthPassword != ""
}

func (c *Config) UseFormAuth() bool {
	return c.FormAuth
}

// URLLabel() exists for backwards compatibility -- prior versions built the URL in the client,
// meaning that the "url" metric label was missing the Port & base path that the XMLConfig provided.
func (c *Config) URLLabel() string {
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

func LoadConfig(flags *flag.FlagSet) (*Config, error) {
	k := koanf.New(".")

	// Defaults
	err := k.Load(confmap.Provider(map[string]interface{}{
		"log-level":   "info",
		"log-format":  "console",
		"api-version": "v3",
		"port":        "8081",
		"interface":   "0.0.0.0",
	}, "."), nil)
	if err != nil {
		return nil, err
	}

	// Environment
	err = k.Load(env.Provider("", ".", func(s string) string {
		s = strings.ToLower(s)
		s = strings.Replace(s, "__", ".", -1)
		s = strings.Replace(s, "_", "-", -1)
		return backwardsCompatibilityTransforms(s)
	}), nil)
	if err != nil {
		return nil, err
	}

	// Flags
	if err = k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		return nil, err
	}

	// XMLConfig
	xmlConfig := k.String("config")
	if xmlConfig != "" {
		err = k.Load(file.Provider(xmlConfig), XMLParser(), koanf.WithMergeFunc(XMLParser().Merge))
		if err != nil {
			return nil, err
		}
	}

	// API Key File
	apiKeyFile := k.String("api-key-file")
	if apiKeyFile != "" {
		data, err := os.ReadFile(apiKeyFile)
		if err != nil {
			return nil, fmt.Errorf("Couldn't Read API Key file %w", err)
		}

		k.Set("api-key", string(data))
	}

	var out Config
	if err := k.Unmarshal("", &out); err != nil {
		return nil, err
	}
	out.k = k
	return &out, nil
}

// ValidateLogLevel validates that the log level is one of the valid log levels
// gookit/Validate is pretty opinionated, and requires that this is not a pointer method.
func (c Config) ValidateLogLevel(val string) bool {
	validLogLevels := []string{}
	for i := zapcore.DebugLevel; i < zapcore.InvalidLevel; i++ {
		validLogLevels = append(validLogLevels, i.String())
	}
	return slices.Contains(validLogLevels, val)

}
func (c *Config) Validate() error {
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

func (c Config) Messages() map[string]string {
	return validate.MS{
		"ApiKey.regex":              "API Key must be a 32 character hex string",
		"LogLevel.validateLogLevel": "Log Level must be one of: debug, info, warn, error, dpanic, panic, fatal",
	}
}

func (c *Config) Translates() map[string]string {
	return validate.MS{
		"LogLevel":                "log-level",
		"LogFormat":               "log-format",
		"URL":                     "url",
		"ApiKey":                  "api-key",
		"ApiKeyFile":              "api-key-file",
		"ApiVersion":              "api-version",
		"XMLConfig":               "config",
		"Port":                    "port",
		"Interface":               "interface",
		"DisableSSLVerify":        "disable-ssl-verify",
		"AuthUsername":            "auth-username",
		"AuthPassword":            "auth-password",
		"FormAuth":                "form-auth",
		"EnableUnknownQueueItems": "enable-unknown-queue-items",
		"EnableAdditionalMetrics": "enable-additional-metric",
	}
}

// Remove in v2.0.0
func backwardsCompatibilityTransforms(s string) string {
	switch s {
	case "apikey-file":
		return "api-key-file"
	case "apikey":
		return "api-key"
	case "basic-auth-username":
		return "auth-username"
	case "basic-auth-password":
		return "auth-password"
	default:
		return s
	}
}
