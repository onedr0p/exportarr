package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-ozzo/ozzo-validation/is"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Arr                     string `koanf:"arr"`
	LogLevel                string `koanf:"log-level"`
	URL                     string `koanf:"url"`
	ApiKey                  string `koanf:"api-key"`
	ApiKeyFile              string `koanf:"api-key-file"`
	ApiVersion              string `koanf:"api-version"`
	XMLConfig               string `koanf:"config"`
	Port                    int    `koanf:"port"`
	Interface               string `koanf:"interface"`
	DisableSSLVerify        bool   `koanf:"disable-ssl-verify"`
	AuthUsername            string `koanf:"auth-username"`
	AuthPassword            string `koanf:"auth-password"`
	FormAuth                bool   `koanf:"form-auth"`
	EnableUnknownQueueItems bool   `koanf:"enable-unknown-queue-items"`
	EnableAdditionalMetrics bool   `koanf:"enable-additional-metric"`
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
		"api-version": "v3",
		"port":        "8081",
		"interface":   "0.0.0.0",
	}, "."), nil)
	if err != nil {
		return nil, err
	}

	// Environment
	err = k.Load(env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", "-", -1)
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

	return &out, nil
}

func (c *Config) Validate() error {
	return validation.ValidateStruct(c,
		validation.Field(&c.URL, validation.Required, is.URL),
		validation.Field(&c.ApiKey,
			validation.Required,
			validation.Match(regexp.MustCompile(`([a-z0-9]{32})`)).
				Error("Invalid API Key, must be 32 characters long and only contain lowercase letters and numbers")),
		validation.Field(&c.Port, validation.Required),
		validation.Field(&c.Interface, validation.Required, is.IP),
		validation.Field(&c.AuthUsername,
			validation.When(c.AuthPassword != "", validation.Required.Error("auth-username is required when auth-password is set")),
			validation.When(c.FormAuth, validation.Required.Error("auth-username is required when form-auth is set"))),
		validation.Field(&c.AuthPassword,
			validation.When(c.AuthUsername != "", validation.Required.Error("auth-password is required when auth-username is set")),
			validation.When(c.FormAuth, validation.Required.Error("auth-password is required when form-auth is set"))),
		validation.Field(&c.ApiVersion, validation.Required, validation.In("v1", "v3")),
	)
}
