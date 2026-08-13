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

func TestArrCommand_BuildCollectors(t *testing.T) {
	params := []struct {
		name          string
		app           arrCommand
		historyToggle bool // when true, also checks DisableHistoryMetrics drops one collector
		wantCount     int
	}{
		{name: "radarr", app: radarrApp, historyToggle: true, wantCount: 7},
		{name: "sonarr", app: sonarrApp, historyToggle: true, wantCount: 7},
		{name: "lidarr", app: lidarrApp, historyToggle: true, wantCount: 7},
		{name: "bazarr", app: bazarrApp, historyToggle: false, wantCount: 1},
		{name: "prowlarr", app: prowlarrApp, historyToggle: true, wantCount: 4},
	}
	for _, p := range params {
		t.Run(p.name, func(t *testing.T) {
			c := &config.ArrConfig{
				URL:    "http://localhost:8080",
				APIKey: "abcdef0123456789abcdef0123456789",
			}
			collectors, err := p.app.buildCollectors(c)
			assert.NoError(t, err)
			assert.Len(t, collectors, p.wantCount)

			if p.historyToggle {
				c.DisableHistoryMetrics = true
				collectors, err = p.app.buildCollectors(c)
				assert.NoError(t, err)
				assert.Len(t, collectors, p.wantCount-1)
			}
		})
	}
}
