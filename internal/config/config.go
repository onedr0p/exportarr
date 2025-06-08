package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/gookit/validate"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

func RegisterConfigFlags(flags *flag.FlagSet) {
	flags.StringP("log-level", "l", "info", "Log level (debug, info, warn, error, fatal, panic)")
	flags.String("log-format", "console", "Log format (console, json)")
	flags.StringP("url", "u", "", "URL to *arr instance")
	flags.StringP("api-key", "a", "", "API Key for *arr instance")
	flags.String("api-key-file", "", "File containing API Key for *arr instance")
	flags.Bool("disable-ssl-verify", false, "Disable SSL verification")
	flags.StringP("interface", "i", "", "IP address to listen on")
	flags.IntP("port", "p", 0, "Port to listen on")
}

type Config struct {
	App              string `koanf:"-"`
	LogLevel         string `koanf:"log-level" validate:"ValidateLogLevel"`
	LogFormat        string `koanf:"log-format" validate:"in:console,json"`
	URL              string `koanf:"url"`
	ApiKey           string `koanf:"api-key"`
	ApiKeyFile       string `koanf:"api-key-file"`
	AuthUsername     string `koanf:"auth-username"`
	AuthPassword     string `koanf:"auth-password"`
	Port             int    `koanf:"port" validate:"required"`
	Interface        string `koanf:"interface" validate:"required|ip"`
	DisableSSLVerify bool   `koanf:"disable-ssl-verify"`
	k                *koanf.Koanf
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
		s = strings.ReplaceAll(s, "__", ".")
		s = strings.ReplaceAll(s, "_", "-")
		return backwardsCompatibilityTransforms(s)
	}), nil)
	if err != nil {
		return nil, err
	}

	// Flags
	if err = k.Load(posflag.Provider(flags, ".", k), nil); err != nil {
		return nil, err
	}

	// API Key File
	apiKeyFile := k.String("api-key-file")
	if apiKeyFile != "" {
		data, err := os.ReadFile(apiKeyFile)
		if err != nil {
			return nil, fmt.Errorf("couldn't Read API Key file %w", err)
		}

		if err := k.Set("api-key", strings.TrimSpace(string(data))); err != nil {
			return nil, fmt.Errorf("couldn't merge api-key into config: %w", err)
		}
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
	return nil
}

func (c Config) Messages() map[string]string {
	return validate.MS{
		"ApiKey.regex":              "api-key must be a 20-32 character alphanumeric string",
		"LogLevel.ValidateLogLevel": "log-level must be one of: debug, info, warn, error, dpanic, panic, fatal",
	}
}

func (c Config) Translates() map[string]string {
	return validate.MS{
		"LogLevel":         "log-level",
		"LogFormat":        "log-format",
		"URL":              "url",
		"ApiKey":           "api-key",
		"ApiKeyFile":       "api-key-file",
		"AuthUsername":     "auth-username",
		"AuthPassword":     "auth-password",
		"ApiVersion":       "api-version",
		"Port":             "port",
		"Interface":        "interface",
		"DisableSSLVerify": "disable-ssl-verify",
	}
}

func (c *Config) UseBasicAuth() bool {
	return c.AuthUsername != "" && c.AuthPassword != ""
}

// Remove in v2.0.0
func backwardsCompatibilityTransforms(s string) string {
	switch s {
	case "apikey-file":
		return "api-key-file"
	case "apikey":
		return "api-key"
	default:
		return s
	}
}
