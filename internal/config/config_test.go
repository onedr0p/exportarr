package config

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"testing"

	"github.com/spf13/pflag"
)

func testFlagSet() *pflag.FlagSet {
	out := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterConfigFlags(out)
	return out
}
func TestLoadConfig_Defaults(t *testing.T) {

	config, err := LoadConfig(&pflag.FlagSet{})
	assert.NoError(t, err)
	assert.Equal(t, config.LogLevel, "info")
	assert.Equal(t, config.LogFormat, "console")
	assert.Equal(t, config.Port, 8081)
	assert.Equal(t, config.Interface, "0.0.0.0")
}

func TestLoadConfig_Flags(t *testing.T) {
	flags := testFlagSet()
	_ = flags.Set("log-level", "debug")
	_ = flags.Set("url", "http://localhost:8989")
	_ = flags.Set("api-key", "abcdef0123456789abcdef0123456789")
	_ = flags.Set("port", "1234")
	_ = flags.Set("interface", "1.2.3.4")
	_ = flags.Set("disable-ssl-verify", "true")
	config, err := LoadConfig(flags)
	assert.NoError(t, err)

	assert.Equal(t, config.LogLevel, "debug")
	assert.Equal(t, config.URL, "http://localhost:8989")
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456789")
	assert.Equal(t, config.Port, 1234)
	assert.Equal(t, config.Interface, "1.2.3.4")
	assert.True(t, config.DisableSSLVerify)
}

func TestLoadConfig_Environment(t *testing.T) {

	// Set environment variables
	t.Setenv("URL", "http://localhost:8989")
	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("PORT", "1234")
	t.Setenv("INTERFACE", "1.2.3.4")
	t.Setenv("DISABLE_SSL_VERIFY", "true")

	config, err := LoadConfig(&pflag.FlagSet{})
	assert.NoError(t, err)

	assert.Equal(t, config.URL, "http://localhost:8989")
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456789")
	assert.Equal(t, config.Port, 1234)
	assert.Equal(t, config.Interface, "1.2.3.4")
	assert.True(t, config.DisableSSLVerify)
}

func TestLoadConfig_PartialEnvironment(t *testing.T) {
	flags := testFlagSet()
	_ = flags.Set("url", "http://localhost:8989")
	_ = flags.Set("interface", "1.2.3.4")

	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456789")
	t.Setenv("PORT", "1234")
	config, err := LoadConfig(flags)
	assert.NoError(t, err)

	// Flags
	assert.Equal(t, config.URL, "http://localhost:8989")
	assert.Equal(t, config.Interface, "1.2.3.4")

	// Env
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456789")
	assert.Equal(t, config.Port, 1234)

	// Defaults
	assert.Equal(t, config.LogLevel, "info")
	assert.Equal(t, config.LogFormat, "console")
}

func TestLoadConfig_OverrideOrder(t *testing.T) {
	flags := testFlagSet()

	// Environment wins over defaults.
	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456781")
	config, err := LoadConfig(flags)
	assert.NoError(t, err)
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456781")

	// Explicitly-set flags win over environment.
	_ = flags.Set("api-key", "abcdef0123456789abcdef0123456780")

	config, err = LoadConfig(flags)
	assert.NoError(t, err)
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456780")
}

func TestLoadConfig_APIKeyFile(t *testing.T) {
	t.Setenv("API_KEY_FILE", "testdata/api_key")

	config, err := LoadConfig(&pflag.FlagSet{})
	assert.NoError(t, err)
	// The fixture ends with a newline, as mounted secrets do: it must be trimmed.
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456783")
}

func TestLoadConfig_APIKeyFile_WinsOverInlineKey(t *testing.T) {
	t.Setenv("API_KEY_FILE", "testdata/api_key")
	t.Setenv("API_KEY", "abcdef0123456789abcdef0123456780")

	config, err := LoadConfig(&pflag.FlagSet{})
	assert.NoError(t, err)
	assert.Equal(t, config.APIKey, "abcdef0123456789abcdef0123456783")
}

func TestLoadConfig_APIKeyFileMissing(t *testing.T) {
	t.Setenv("API_KEY_FILE", "testdata/does_not_exist")

	_, err := LoadConfig(&pflag.FlagSet{})
	assert.Error(t, err)
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
				LogFormat: "console",
				URL:       "http://localhost",
				APIKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0.0",
			},
		},
		{
			name: "missing-port",
			config: &Config{
				LogLevel:  "debug",
				LogFormat: "console",
				URL:       "http://localhost",
				APIKey:    "abcdef0123456789abcdef0123456789",
				Port:      0,
				Interface: "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-interface",
			config: &Config{
				LogLevel:  "debug",
				LogFormat: "console",
				URL:       "http://localhost",
				APIKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-log-level",
			config: &Config{
				LogLevel:  "asdf",
				LogFormat: "console",
				URL:       "http://localhost",
				APIKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0.0",
			},
			shouldError: true,
		},
		{
			name: "bad-log-format",
			config: &Config{
				LogLevel:  "debug",
				LogFormat: "yaml",
				URL:       "http://localhost",
				APIKey:    "abcdef0123456789abcdef0123456789",
				Port:      1234,
				Interface: "0.0.0.0",
			},
			shouldError: true,
		},
	}

	for _, p := range parameters {
		t.Run(p.name, func(t *testing.T) {

			err := p.config.Validate()
			if p.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
