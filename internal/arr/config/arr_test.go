package config

import (
	"testing"

	base_config "github.com/shamelin/exportarr/internal/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func testFlagSet() *pflag.FlagSet {
	ret := pflag.NewFlagSet("test", pflag.ContinueOnError)
	RegisterArrFlags(ret)
	return ret
}

func TestUsecAuth(t *testing.T) {
	c := ArrConfig{
		AuthUsername: "user",
		AuthPassword: "pass",
	}
	require := require.New(t)
	require.True(c.UseBasicAuth())
	require.False(c.UseFormAuth())
	c.FormAuth = true
	require.True(c.UseFormAuth())
	require.False(c.UseBasicAuth())

}

func TestBaseURL(t *testing.T) {
	c := ArrConfig{
		URL:        "http://localhost:8080",
		ApiVersion: "v1",
	}
	require := require.New(t)
	require.Equal("http://localhost:8080/api/v1", c.BaseURL())
}

func TestLoadConfig_Defaults(t *testing.T) {
	flags := testFlagSet()
	c := base_config.Config{
		URL:              "http://localhost",
		ApiKey:           "abcdef0123456789abcdef0123456789",
		DisableSSLVerify: true,
	}

	require := require.New(t)

	config, err := LoadArrConfig(c, flags)
	require.NoError(err)

	require.Equal("v3", config.ApiVersion)

	// base config values are not overwritten
	require.Equal("http://localhost", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.True(config.DisableSSLVerify)
}

func TestLoadConfig_Environment(t *testing.T) {
	flags := testFlagSet()
	c := base_config.Config{
		URL:              "http://localhost",
		ApiKey:           "abcdef0123456789abcdef0123456789",
		DisableSSLVerify: true,
	}

	require := require.New(t)
	t.Setenv("AUTH_USERNAME", "user")
	t.Setenv("AUTH_PASSWORD", "pass")
	t.Setenv("FORM_AUTH", "true")
	t.Setenv("ENABLE_UNKNOWN_QUEUE_ITEMS", "true")
	t.Setenv("ENABLE_ADDITIONAL_METRICS", "true")

	config, err := LoadArrConfig(c, flags)
	require.NoError(err)

	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
	require.True(config.FormAuth)
	require.True(config.EnableUnknownQueueItems)
	require.True(config.EnableAdditionalMetrics)

	// defaults are not overwritten
	require.Equal("v3", config.ApiVersion)

	// base config values are not overwritten
	require.Equal("http://localhost", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
	require.True(config.DisableSSLVerify)

}

func TestLoadConfig_PartialEnvironment(t *testing.T) {
	flags := testFlagSet()
	flags.Set("auth-username", "user")
	flags.Set("auth-password", "pass")

	t.Setenv("ENABLE_UNKNOWN_QUEUE_ITEMS", "true")
	t.Setenv("ENABLE_ADDITIONAL_METRICS", "true")

	c := base_config.Config{
		URL:    "http://localhost",
		ApiKey: "abcdef0123456789abcdef0123456789",
	}

	require := require.New(t)
	config, err := LoadArrConfig(c, flags)
	require.NoError(err)

	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
	require.True(config.EnableUnknownQueueItems)
	require.True(config.EnableAdditionalMetrics)

	require.Equal("http://localhost", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)

	require.Equal("v3", config.ApiVersion)

}

func TestLoadConfig_Flags(t *testing.T) {
	flags := testFlagSet()
	flags.Set("auth-username", "user")
	flags.Set("auth-password", "pass")
	flags.Set("form-auth", "true")
	flags.Set("enable-unknown-queue-items", "true")
	flags.Set("enable-additional-metrics", "true")
	c := base_config.Config{}

	// should be overridden by flags
	t.Setenv("AUTH_USERNAME", "user2")

	require := require.New(t)
	config, err := LoadArrConfig(c, flags)
	require.NoError(err)
	require.Equal("user", config.AuthUsername)
	require.Equal("pass", config.AuthPassword)
	require.True(config.FormAuth)
	require.True(config.EnableUnknownQueueItems)
	require.True(config.EnableAdditionalMetrics)

	// defaults fall through
	require.Equal("v3", config.ApiVersion)
}

func TestLoadConfig_XMLConfig(t *testing.T) {
	flags := testFlagSet()
	flags.Set("config", "test_fixtures/config.test_xml")
	c := base_config.Config{
		URL: "http://localhost",
	}

	config, err := LoadArrConfig(c, flags)

	require := require.New(t)
	require.NoError(err)

	// schema/host from config, port, and asdf from xml, api & version defaulted in LoadConfig.
	require.Equal("http://localhost:7878/asdf", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
}

func TestLoadConfig_XMLConfigEnv(t *testing.T) {
	flags := testFlagSet()
	t.Setenv("CONFIG", "test_fixtures/config.test_xml")
	c := base_config.Config{
		URL: "http://localhost",
	}

	config, err := LoadArrConfig(c, flags)

	require := require.New(t)
	require.NoError(err)

	// schema/host from config, port, and asdf from xml, api & version defaulted in LoadConfig.
	require.Equal("http://localhost:7878/asdf", config.URL)
	require.Equal("abcdef0123456789abcdef0123456789", config.ApiKey)
}

func TestValidate(t *testing.T) {
	params := []struct {
		name   string
		config *ArrConfig
		valid  bool
	}{
		{
			name: "good-basic-auth",
			config: &ArrConfig{
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				AuthUsername: "user",
				AuthPassword: "pass",
			},
			valid: true,
		},
		{
			name: "good-form-auth",
			config: &ArrConfig{
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				AuthUsername: "user",
				AuthPassword: "pass",
				FormAuth:     true,
			},
			valid: true,
		},
		{
			name: "good-api-key-32-len",
			config: &ArrConfig{
				URL:        "http://localhost",
				ApiKey:     "abcdefABCDEF0123456789abcdef0123",
				ApiVersion: "v3",
			},
			valid: true,
		},
		{
			name: "good-api-key-32-len",
			config: &ArrConfig{
				URL:        "http://localhost",
				ApiKey:     "abcdefABCDEF01234567",
				ApiVersion: "v3",
			},
			valid: true,
		},
		{
			name: "bad-api-key",
			config: &ArrConfig{
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abc",
				ApiVersion: "v3",
			},
			valid: false,
		},
		{
			name: "no-api-version",
			config: &ArrConfig{
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "",
			},
			valid: true,
		},
		{
			name: "password-needs-username",
			config: &ArrConfig{
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				AuthPassword: "password",
			},
			valid: false,
		},
		{
			name: "username-needs-password",
			config: &ArrConfig{
				URL:          "http://localhost",
				ApiKey:       "abcdef0123456789abcdef0123456789",
				ApiVersion:   "v3",
				AuthUsername: "username",
			},
			valid: false,
		},
		{
			name: "form-auth-needs-user-and-password",
			config: &ArrConfig{
				URL:        "http://localhost",
				ApiKey:     "abcdef0123456789abcdef0123456789",
				ApiVersion: "v3",
				FormAuth:   true,
			},
			valid: false,
		},
	}
	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			require := require.New(t)
			err := p.config.Validate()
			if p.valid {
				require.NoError(err)
			} else {
				require.Error(err)
			}
		})
	}
}
