package config

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func testFlagSet() *pflag.FlagSet {
	out := pflag.NewFlagSet("test", pflag.ContinueOnError)
	out.StringP("log-level", "l", "info", "Log level (debug, info, warn, error, fatal, panic)")
	out.StringP("config", "c", "", "*arr config.xml file for parsing authentication information")
	out.StringP("url", "u", "", "URL to *arr instance")
	out.StringP("api-key", "k", "", "API Key for *arr instance")
	out.StringP("api-key-file", "f", "", "File containing API Key for *arr instance")
	out.Int("port", 0, "Port to listen on")
	out.StringP("interface", "i", "", "IP address to listen on")
	out.Bool("disable-ssl-verify", false, "Disable SSL verification")
	out.String("auth-username", "", "Username for basic auth")
	out.String("auth-password", "", "Password for basic auth")
	out.Bool("form-auth", false, "Use form based authentication")
	out.Bool("enable-unknown-queue-items", false, "Enable unknown queue items")
	out.Bool("enable-additional-metric", false, "Enable additional metric")
	return out
}
func TestLoadConfig_Defaults(t *testing.T) {
	require := require.New(t)

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)
	require.Equal("info", config.LogLevel)
	require.Equal("console", config.LogFormat)
	require.Equal("v3", config.ApiVersion)
	require.Equal(8081, config.Port)
	require.Equal("0.0.0.0", config.Interface)
}

func TestLoadConfig_Flags(t *testing.T) {
	flags := testFlagSet()
	flags.Set("log-level", "debug")
	flags.Set("url", "http://localhost:8989")
	flags.Set("api-key", "abcdef0123456789abcdef0123456789")
	flags.Set("port", "1234")
	flags.Set("interface", "1.2.3.4")
	flags.Set("disable-ssl-verify", "true")
	flags.Set("auth-username", "user")
	flags.Set("auth-password", "pass")
	flags.Set("form-auth", "true")
	flags.Set("enable-unknown-queue-items", "true")
	flags.Set("enable-additional-metric", "true")

	require := require.New(t)
	config, err := LoadConfig(flags)
	require.NoError(err)

	require.Equal("debug", config.LogLevel)
	require.Equal("http://localhost:8989", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.Equal(1234, config.Port)
	require.Equal("1.2.3.4", config.Interface)
	require.True(config.DisableSSLVerify)
	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
	require.True(config.FormAuth)
	require.True(config.EnableUnknownQueueItems)
	require.True(config.EnableAdditionalMetrics)
	// Defaults fall through
	require.Equal("v3", config.ApiVersion)
	require.True(config.UseFormAuth())
	require.False(config.UseBasicAuth())

	flags.Set("form-auth", "false")
	config, err = LoadConfig(flags)
	require.NoError(err)
	require.False(config.UseFormAuth())
	require.True(config.UseBasicAuth())
}

func TestLoadConfig_Environment(t *testing.T) {
	require := require.New(t)

	// Set environment variables
	t.Setenv("URL", "http://localhost:8989")
	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("PORT", "1234")
	t.Setenv("INTERFACE", "1.2.3.4")
	t.Setenv("DISABLE_SSL_VERIFY", "true")
	t.Setenv("AUTH_USERNAME", "user")
	t.Setenv("AUTH_PASSWORD", "pass")
	t.Setenv("FORM_AUTH", "true")
	t.Setenv("ENABLE_UNKNOWN_QUEUE_ITEMS", "true")
	t.Setenv("ENABLE_ADDITIONAL_METRIC", "true")

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)

	require.Equal("http://localhost:8989", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.Equal(1234, config.Port)
	require.Equal("1.2.3.4", config.Interface)
	require.True(config.DisableSSLVerify)
	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
	require.True(config.FormAuth)
	require.True(config.EnableUnknownQueueItems)
	require.True(config.EnableAdditionalMetrics)
	// Defaults fall through
	require.Equal("v3", config.ApiVersion)
}

func TestLoadConfig_BackwardsCompatibility(t *testing.T) {
	require := require.New(t)

	// Set environment variables
	t.Setenv("URL", "http://localhost:8989")
	t.Setenv("APIKEY_FILE", "./test_fixtures/api_key")
	t.Setenv("PORT", "1234")
	t.Setenv("BASIC_AUTH_USERNAME", "user")
	t.Setenv("BASIC_AUTH_PASSWORD", "pass")

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)

	require.Equal("abcdef0123456789abcdef0123456783", config.ApiKey)
	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
}

func TestLoadConfig_XMLConfig(t *testing.T) {
	flags := testFlagSet()
	flags.Set("config", "test_fixtures/config.test_xml")
	flags.Set("url", "http://localhost")

	require := require.New(t)
	config, err := LoadConfig(flags)
	require.NoError(err)

	require.Equal("http://localhost:7878/asdf", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
}

func TestLoadConfig_ApiKeyFile(t *testing.T) {
	flags := testFlagSet()
	flags.Set("api-key-file", "test_fixtures/api_key")

	require := require.New(t)
	config, err := LoadConfig(flags)
	require.NoError(err)

	require.Equal("abcdef0123456789abcdef0123456783", config.ApiKey)
}

func TestLoadConfig_OverrideOrder(t *testing.T) {

	require := require.New(t)
	flags := testFlagSet()

	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456781")
	config, err := LoadConfig(flags)
	require.NoError(err)
	require.Equal("abcdef0123456789abcdef0123456781", config.ApiKey)

	flags.Set("api-key", "abcdef0123456789abcdef0123456780")

	config, err = LoadConfig(flags)
	require.NoError(err)
	require.Equal("abcdef0123456789abcdef0123456780", config.ApiKey)

	flags.Set("config", "test_fixtures/config.test_xml")
	config, err = LoadConfig(flags)
	require.NoError(err)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)

	flags.Set("api-key-file", "test_fixtures/api_key")
	config, err = LoadConfig(flags)
	require.NoError(err)
	require.Equal("abcdef0123456789abcdef0123456783", config.ApiKey)
}

func TestValidate(t *testing.T) {
	parameters := []struct {
		name        string
		config      *Config
		shouldError bool
	}{
		{
			name: "good",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				Port:       1234,
				Interface:  "0.0.0.0",
			},
		},
		{
			name: "good-basic-auth",
			config: &Config{
				LogLevel:     "debug",
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				Port:         1234,
				Interface:    "0.0.0.0",
				AuthUsername: "user",
				AuthPassword: "pass",
			},
		},
		{
			name: "good-form-auth",
			config: &Config{
				LogLevel:     "debug",
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				Port:         1234,
				Interface:    "0.0.0.0",
				AuthUsername: "user",
				AuthPassword: "pass",
				FormAuth:     true,
			},
		},
		{
			name: "bad-api-key",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef01234567",
				ApiVersion: "v3",
				Port:       1234,
				Interface:  "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-api-version",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v2",
				Port:       1234,
				Interface:  "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "missing-port",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				Port:       0,
				Interface:  "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-interface",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				Port:       1234,
				Interface:  "0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-log-level",
			config: &Config{
				LogLevel:   "asdf",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				Port:       1234,
				Interface:  "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "password-needs-username",
			config: &Config{
				LogLevel:     "debug",
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				Port:         1234,
				Interface:    "0.0.0.0",
				AuthPassword: "password",
			},
			shouldError: true,
		},
		{
			name: "username-needs-password",
			config: &Config{
				LogLevel:     "debug",
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				Port:         1234,
				Interface:    "0.0.0.0",
				AuthUsername: "username",
			},
			shouldError: true,
		},
		{
			name: "form-auth-needs-user-and-password",
			config: &Config{
				LogLevel:   "debug",
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				Port:       1234,
				Interface:  "0.0.0.0",
				FormAuth:   true,
			},
			shouldError: true,
		},
	}

	for _, p := range parameters {
		t.Run(p.name, func(t *testing.T) {
			require := require.New(t)

			err := p.config.Validate()
			if p.shouldError {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}
