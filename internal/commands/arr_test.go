package commands

import (
	"testing"

	"github.com/onedr0p/exportarr/internal/arr/config"
	base_config "github.com/onedr0p/exportarr/internal/config"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestBackwardsCompatibility(t *testing.T) {
	params := []struct {
		name  string
		flags *pflag.FlagSet
	}{
		{
			name:  "radarr",
			flags: radarrCmd.PersistentFlags(),
		},
		{
			name:  "sonarr",
			flags: sonarrCmd.PersistentFlags(),
		},
		{
			name:  "lidarr",
			flags: lidarrCmd.PersistentFlags(),
		},
		{
			name:  "readarr",
			flags: readarrCmd.PersistentFlags(),
		},
		{
			name:  "prowlarr",
			flags: prowlarrCmd.PersistentFlags(),
		},
		{
			name:  "bazarr",
			flags: bazarrCmd.PersistentFlags(),
		},
	}
	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			p.flags.Set("basic-auth-username", "user")
			p.flags.Set("basic-auth-password", "pass")

			require := require.New(t)
			config, err := config.LoadArrConfig(base_config.Config{}, p.flags)
			require.NoError(err)
			require.Equal("user", config.AuthUsername)
			require.Equal("pass", config.AuthPassword)
		})
	}

}
