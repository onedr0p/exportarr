package commands

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"testing"

	"github.com/onedr0p/exportarr/internal/arr/config"
	base_config "github.com/onedr0p/exportarr/internal/config"
	"github.com/spf13/pflag"
)

func TestAuthFlagsRegistered(t *testing.T) {
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
			_ = p.flags.Set("auth-username", "user")
			_ = p.flags.Set("auth-password", "pass")
			config, err := config.LoadArrConfig(base_config.Config{}, p.flags)
			assert.NoError(t, err)
			assert.Equal(t, config.AuthUsername, "user")
			assert.Equal(t, config.AuthPassword, "pass")
		})
	}

}
