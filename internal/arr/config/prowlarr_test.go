package config

import (
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestLoadProwlarrConfig(t *testing.T) {
	flags := pflag.FlagSet{}
	RegisterProwlarrFlags(&flags)

	flags.Set("backfill", "true")
	flags.Set("backfill-since-date", "2021-01-01")
	c := ArrConfig{
		URL:              "http://localhost",
		ApiKey:           "abcdef0123456789abcdef0123456789",
		DisableSSLVerify: true,
		k:                koanf.New("."),
	}
	c.LoadProwlarrConfig(&flags)

	require := require.New(t)
	require.True(c.Prowlarr.Backfill)
	require.Equal("2021-01-01", c.Prowlarr.BackfillSinceDate)
	require.Equal("2021-01-01", c.Prowlarr.BackfillSinceTime.Format("2006-01-02"))
	require.Equal("http://localhost", c.URL)
	require.Equal("abcdef0123456789abcdef0123456789", c.ApiKey)
	require.True(c.DisableSSLVerify)
}

func TestValidateProwlarr(t *testing.T) {
	tm, _ := time.Parse("2006-01-02", "2021-01-01")
	parameters := []struct {
		name        string
		config      *ProwlarrConfig
		shouldError bool
	}{
		{
			name: "good",
			config: &ProwlarrConfig{
				Backfill:          true,
				BackfillSinceTime: tm,
				BackfillSinceDate: "2021-01-01",
			},
		},
		{
			name: "bad-date",
			config: &ProwlarrConfig{
				Backfill:          true,
				BackfillSinceTime: tm,
				BackfillSinceDate: "2021-31-31",
			},
			shouldError: true,
		},
	}

	for _, parameter := range parameters {
		t.Run(parameter.name, func(t *testing.T) {
			err := parameter.config.Validate()
			if parameter.shouldError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
