package test_util

import (
	"os"

	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func NewTestServer(t *testing.T, fixture_dir string, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
		require.NotEmpty(t, r.URL.Path)
		// turns /api/some/path into some_path
		endpoint := strings.Replace(strings.Replace(r.URL.Path, "/api/", "", -1), "/", "_", -1)
		w.WriteHeader(http.StatusOK)
		// NOTE: this assumes there is a file that matches the some_path
		json, err := os.ReadFile(filepath.Join(fixture_dir, endpoint+".json"))
		require.NoError(t, err)
		_, err = w.Write(json)
		require.NoError(t, err)
	})), nil
}

func NewTestSharedServer(t *testing.T, fn func(http.ResponseWriter, *http.Request)) (*httptest.Server, error) {
	return NewTestServer(t, COMMON_FIXTURES_PATH, fn)
}
