package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onedr0p/exportarr/internal/assert"
)

func TestLoadSabnzbdInstanceEnv_Unconfigured(t *testing.T) {
	inst, err := LoadSabnzbdInstanceEnv("SABNZBD__")
	assert.NoError(t, err)
	assert.False(t, inst.Configured())
	assert.Empty(t, inst.URL)
}

func TestLoadSabnzbdInstanceEnv_Basic(t *testing.T) {
	t.Setenv("SABNZBD__URL", "http://sabnzbd.local:8080")
	t.Setenv("SABNZBD__API_KEY", "abcdef0123456789")
	t.Setenv("SABNZBD__DISABLE_SSL_VERIFY", "true")
	t.Setenv("SABNZBD__REQUEST_TIMEOUT", "5s")

	inst, err := LoadSabnzbdInstanceEnv("SABNZBD__")
	assert.NoError(t, err)
	assert.True(t, inst.Configured())
	assert.Equal(t, inst.URL, "http://sabnzbd.local:8080")
	assert.Equal(t, inst.APIKey, "abcdef0123456789")
	assert.True(t, inst.DisableSSLVerify)
	assert.Equal(t, inst.RequestTimeout, 5*time.Second)
}

func TestLoadSabnzbdInstanceEnv_RequestTimeoutDefault(t *testing.T) {
	t.Setenv("SABNZBD__URL", "http://sabnzbd.local:8080")
	inst, err := LoadSabnzbdInstanceEnv("SABNZBD__")
	assert.NoError(t, err)
	assert.Equal(t, inst.RequestTimeout, 60*time.Second)
}

func TestLoadSabnzbdInstanceEnv_APIKeyFileOverridesInline(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "sabnzbd_api_key")
	assert.NoError(t, os.WriteFile(keyFile, []byte("fromfile0123456789\n"), 0o600))

	t.Setenv("SABNZBD__URL", "http://sabnzbd.local:8080")
	t.Setenv("SABNZBD__API_KEY", "inlinekey0123456789")
	t.Setenv("SABNZBD__API_KEY_FILE", keyFile)

	inst, err := LoadSabnzbdInstanceEnv("SABNZBD__")
	assert.NoError(t, err)
	assert.Equal(t, inst.APIKey, "fromfile0123456789")
}

func TestNewSabnzbdConfigForInstance(t *testing.T) {
	inst := SabnzbdInstanceEnv{
		URL:              "http://sabnzbd.local:8080",
		APIKey:           "abcdef0123456789",
		DisableSSLVerify: true,
		RequestTimeout:   5 * time.Second,
	}

	c := NewSabnzbdConfigForInstance(inst)

	assert.Equal(t, c.URL, "http://sabnzbd.local:8080")
	assert.Equal(t, c.APIKey, "abcdef0123456789")
	assert.True(t, c.DisableSSLVerify)
	assert.Equal(t, c.RequestTimeout, 5*time.Second)
}
