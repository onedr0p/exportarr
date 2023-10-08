package config

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func testFlagSet() *pflag.FlagSet {
	out := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterConfigFlags(out)
	return out
}
func TestLoadConfig_Defaults(t *testing.T) {
	require := require.New(t)

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)
	require.Equal("info", config.LogLevel)
	require.Equal("console", config.LogFormat)
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

	require := require.New(t)
	config, err := LoadConfig(flags)
	require.NoError(err)

	require.Equal("debug", config.LogLevel)
	require.Equal("http://localhost:8989", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.Equal(1234, config.Port)
	require.Equal("1.2.3.4", config.Interface)
	require.True(config.DisableSSLVerify)

	flags.Set("form-auth", "false")
	_, err = LoadConfig(flags)
	require.NoError(err)
}

func TestLoadConfig_Environment(t *testing.T) {
	require := require.New(t)

	// Set environment variables
	t.Setenv("URL", "http://localhost:8989")
	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("PORT", "1234")
	t.Setenv("INTERFACE", "1.2.3.4")
	t.Setenv("DISABLE_SSL_VERIFY", "true")

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)

	require.Equal("http://localhost:8989", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.Equal(1234, config.Port)
	require.Equal("1.2.3.4", config.Interface)
	require.True(config.DisableSSLVerify)
}

func TestLoadConfig_PartialEnvironment(t *testing.T) {
	flags := testFlagSet()
	flags.Set("url", "http://localhost:8989")
	flags.Set("interface", "1.2.3.4")

	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("PORT", "1234")

	require := require.New(t)
	config, err := LoadConfig(flags)
	require.NoError(err)

	// Env
	require.Equal("http://localhost:8989", config.URL)
	require.Equal("1.2.3.4", config.Interface)

	// Flags
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.Equal(1234, config.Port)

	// Defaults
	require.Equal("info", config.LogLevel)
	require.Equal("console", config.LogFormat)
}

func TestLoadConfig_BackwardsCompatibility_ApiKeyFile(t *testing.T) {
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
}

func TestLoadConfig_BackwardsCompatibility_ApiKey(t *testing.T) {
	require := require.New(t)

	// Set environment variables
	t.Setenv("URL", "http://localhost:8989")
	t.Setenv("APIKEY", "abcdef0123456789abcdef0123456780")
	t.Setenv("PORT", "1234")

	config, err := LoadConfig(&pflag.FlagSet{})
	require.NoError(err)

	require.Equal("abcdef0123456789abcdef0123456780", config.ApiKey)
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
				LogLevel:  "debug",
				URL:       "http://localhost",
				ApiKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0.0",
			},
		},
		{
			name: "missing-port",
			config: &Config{
				LogLevel:  "debug",
				URL:       "http://localhost",
				ApiKey:    "abcdef0123456789abcdef0123456789",
				Port:      0,
				Interface: "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-interface",
			config: &Config{
				LogLevel:  "debug",
				URL:       "http://localhost",
				ApiKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-log-level",
			config: &Config{
				LogLevel:  "asdf",
				URL:       "http://localhost",
				ApiKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0.0",
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
