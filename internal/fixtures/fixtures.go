package fixtures

import (
	"os"

	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onedr0p/exportarr/internal/assert"
)

// NewTestServer serves JSON fixtures from fixtureDir keyed by request path,
// invoking fn for per-request assertions.
func NewTestServer(t *testing.T, fixtureDir string, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
		assert.NotEmpty(t, r.URL.Path)
		// turns /api/some/path into some_path
		endpoint := strings.ReplaceAll(strings.ReplaceAll(r.URL.Path, "/api/", ""), "/", "_")
		w.WriteHeader(http.StatusOK)
		// NOTE: this assumes there is a file that matches the some_path
		json, err := os.ReadFile(filepath.Join(fixtureDir, endpoint+".json")) //nolint:gosec // test fixture path
		assert.NoError(t, err)
		_, err = w.Write(json) //nolint:gosec // fixture bytes, not user input
		assert.NoError(t, err)
	})), nil
}

// NewTestSharedServer is NewTestServer rooted at the common fixture dir.
func NewTestSharedServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return NewTestServer(t, CommonFixturesPath, fn)
}
