package collector

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	client "github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/fixtures"
	"github.com/prometheus/client_golang/prometheus"
)

// fixtureServer serves JSON fixtures from the app's fixture dir, falling back
// to the shared common dir — mirroring how a real instance answers every
// endpoint a command's collector set hits in one scrape.
func fixtureServer(t *testing.T, appDir string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := strings.ReplaceAll(strings.TrimPrefix(r.URL.Path, "/api/"), "/", "_")
		for _, dir := range []string{appDir, fixtures.CommonFixturesPath} {
			b, err := os.ReadFile(filepath.Join(dir, endpoint+".json")) //nolint:gosec // test fixture path
			if err != nil {
				continue
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(b) //nolint:gosec // fixture bytes, not user input
			return
		}
		t.Errorf("no fixture for endpoint %q", endpoint)
		w.WriteHeader(http.StatusNotFound)
	}))
}

// TestPerAppCollectorSets registers the exact collector set each command wires
// up and gathers it against fixture-backed responses: one scrape must succeed
// end-to-end with no collector errors and no descriptor collisions.
func TestPerAppCollectorSets(t *testing.T) {
	apps := []struct {
		name       string
		apiVersion string
		fixtures   string
		primary    func(*client.Client, *config.ArrConfig) prometheus.Collector
	}{
		{"radarr", "v3", "../testdata/radarr/", NewRadarrCollector},
		{"sonarr", "v3", "../testdata/sonarr/", NewSonarrCollector},
		{"lidarr", "v3", "../testdata/lidarr/", NewLidarrCollector},
	}

	for _, app := range apps {
		t.Run(app.name, func(t *testing.T) {
			ts := fixtureServer(t, app.fixtures)
			defer ts.Close()

			conf := &config.ArrConfig{
				App:        app.name,
				APIVersion: app.apiVersion,
				URL:        ts.URL,
				APIKey:     fixtures.APIKey,
			}
			cl, err := client.NewClient(conf)
			assert.NoError(t, err)

			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(
				app.primary(cl, conf),
				NewQueueCollector(cl, conf),
				NewHistoryCollector(cl, conf),
				NewRootFolderCollector(cl, conf),
				NewDiskSpaceCollector(cl, conf),
				NewSystemStatusCollector(cl, conf),
				NewSystemHealthCollector(cl, conf),
			)

			families, err := registry.Gather()
			assert.NoError(t, err, "gathering the full collector set should not error")

			names := make(map[string]bool, len(families))
			for _, mf := range families {
				names[mf.GetName()] = true
			}
			assert.True(t, names[app.name+"_diskspace_free_bytes"], "diskspace free metric missing")
			assert.True(t, names[app.name+"_diskspace_total_bytes"], "diskspace total metric missing")
			assert.True(t, names[app.name+"_queue_total"], "queue metric missing")
			for name := range names {
				assert.NotContains(t, name, "_collector_error",
					"no collector should report an error against complete fixtures")
			}
		})
	}
}
