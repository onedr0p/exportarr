package commands

import (
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/onedr0p/exportarr/internal/assert"
)

func validRadarrEnv(t *testing.T) {
	t.Helper()
	t.Setenv("RADARR__URL", "http://radarr.local:7878")
	t.Setenv("RADARR__API_KEY", "abcdef0123456789abcdef0123456789")
}

func validSabnzbdEnv(t *testing.T) {
	t.Helper()
	t.Setenv("SABNZBD__URL", "http://sabnzbd.local:8080")
	t.Setenv("SABNZBD__API_KEY", "abcdef0123456789")
}

func TestBuildMultiTargets_NoAppsConfigured(t *testing.T) {
	_, err := buildMultiTargets()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RADARR__URL")
	assert.Contains(t, err.Error(), "SABNZBD__URL")
}

func TestBuildMultiTargets_OneApp(t *testing.T) {
	validRadarrEnv(t)

	targets, err := buildMultiTargets()
	assert.NoError(t, err)
	assert.Len(t, targets, 1)
	assert.Equal(t, targets[0].path, "radarr")
	assert.Equal(t, targets[0].app, "radarr")
	assert.Equal(t, targets[0].url, "http://radarr.local:7878")

	// register populates the same collector count as the standalone radarr
	// subcommand (verified in TestArrCommand_BuildCollectors)
	reg := prometheus.NewRegistry()
	assert.NotPanics(t, func() { targets[0].register(reg) })
}

func TestBuildMultiTargets_MultipleApps(t *testing.T) {
	validRadarrEnv(t)
	validSabnzbdEnv(t)

	targets, err := buildMultiTargets()
	assert.NoError(t, err)
	assert.Len(t, targets, 2)

	paths := map[string]bool{}
	for _, tg := range targets {
		paths[tg.path] = true
	}
	assert.True(t, paths["radarr"])
	assert.True(t, paths["sabnzbd"])
}

func TestBuildMultiTargets_InvalidInstanceReportsAppName(t *testing.T) {
	t.Setenv("RADARR__URL", "http://radarr.local:7878")
	t.Setenv("RADARR__API_KEY", "too-short")

	_, err := buildMultiTargets()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "radarr")
}

func TestBuildMultiTargets_MountsThroughServeHTTPMux(t *testing.T) {
	validRadarrEnv(t)

	targets, err := buildMultiTargets()
	assert.NoError(t, err)

	// Resolve routing only (mux.Handler, not ServeHTTP) so this doesn't
	// trigger a real scrape against the (nonexistent) configured URL.
	mux := buildMux(targets)
	_, pattern := mux.Handler(httptest.NewRequest("GET", "/metrics/radarr", nil))
	assert.Equal(t, pattern, "/metrics/radarr")
}
