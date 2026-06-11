// Package config loads *arr-specific exportarr configuration.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/caarlos0/env/v11"
	flag "github.com/spf13/pflag"

	base_config "github.com/onedr0p/exportarr/internal/config"
)

// apiKeyRegex matches the *arr API key format.
var apiKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9]{20,32}$`)

// RegisterArrFlags registers the *arr-specific flags on the given FlagSet.
func RegisterArrFlags(flags *flag.FlagSet) {
	flags.String("auth-username", "", "Username for form auth")
	flags.String("auth-password", "", "Password for form auth")
	flags.Bool("form-auth", false, "Use form based authentication")
	flags.Bool("enable-unknown-queue-items", false, "Enable unknown queue items")
	flags.Bool("disable-quality-metrics", false, "Skip per-item quality breakdowns (episodefile/trackfile + qualitydefinition lookups; ~1 API call per series/artist each scrape)")
	flags.Bool("disable-episode-metrics", false, "Skip per-episode metrics (sonarr episode monitoring lookups, bazarr episode-subtitle walk; load scales with library size)")
	flags.Bool("disable-album-metrics", false, "Skip per-album metrics (lidarr album lookups; ~1 API call per artist each scrape)")
	flags.Bool("disable-history-metrics", false, "Skip the history endpoint; its total forces a full count over the (unprunable) history table, which is slow on multi-year instances")
	flags.Bool("disable-wanted-metrics", false, "Skip the wanted/missing and wanted/cutoff endpoints; their totals force full counts, which is slow on very large libraries")
}

// ArrConfig is the configuration for an *arr exporter.
type ArrConfig struct {
	App                     string         `env:"-"`
	APIVersion              string         `env:"API_VERSION" envDefault:"v3"`
	AuthUsername            string         `env:"AUTH_USERNAME"`
	AuthPassword            string         `env:"AUTH_PASSWORD"`
	FormAuth                bool           `env:"FORM_AUTH"`
	EnableUnknownQueueItems bool           `env:"ENABLE_UNKNOWN_QUEUE_ITEMS"`
	DisableQualityMetrics   bool           `env:"DISABLE_QUALITY_METRICS"`
	DisableEpisodeMetrics   bool           `env:"DISABLE_EPISODE_METRICS"`
	DisableAlbumMetrics     bool           `env:"DISABLE_ALBUM_METRICS"`
	DisableHistoryMetrics   bool           `env:"DISABLE_HISTORY_METRICS"`
	DisableWantedMetrics    bool           `env:"DISABLE_WANTED_METRICS"`
	URL                     string         `env:"-"` // from the base config
	APIKey                  string         `env:"-"` // from the base config
	DisableSSLVerify        bool           `env:"-"` // from the base config
	RequestTimeout          time.Duration  `env:"-"` // from the base config
	Prowlarr                ProwlarrConfig `envPrefix:"PROWLARR__"`
	Bazarr                  BazarrConfig   `envPrefix:"BAZARR__"`
}

// UseFormAuth reports whether form-based authentication is enabled.
func (c *ArrConfig) UseFormAuth() bool {
	return c.FormAuth
}

// BaseURL returns the *arr API base URL including the API version.
func (c *ArrConfig) BaseURL() string {
	ret, _ := url.JoinPath(c.URL, "api", c.APIVersion)
	return ret
}

// LoadArrConfig parses environment variables into an ArrConfig seeded from the
// base configuration, then overlays any explicitly-set flags.
func LoadArrConfig(conf base_config.Config, flags *flag.FlagSet) (*ArrConfig, error) {
	out := &ArrConfig{
		App:              conf.App,
		URL:              conf.URL,
		APIKey:           conf.APIKey,
		DisableSSLVerify: conf.DisableSSLVerify,
		RequestTimeout:   conf.RequestTimeout,
	}
	if err := env.Parse(out); err != nil {
		return nil, err
	}

	base_config.OverlayFlag(flags, "auth-username", flags.GetString, &out.AuthUsername)
	base_config.OverlayFlag(flags, "auth-password", flags.GetString, &out.AuthPassword)
	base_config.OverlayFlag(flags, "form-auth", flags.GetBool, &out.FormAuth)
	base_config.OverlayFlag(flags, "enable-unknown-queue-items", flags.GetBool, &out.EnableUnknownQueueItems)
	base_config.OverlayFlag(flags, "disable-quality-metrics", flags.GetBool, &out.DisableQualityMetrics)
	base_config.OverlayFlag(flags, "disable-episode-metrics", flags.GetBool, &out.DisableEpisodeMetrics)
	base_config.OverlayFlag(flags, "disable-album-metrics", flags.GetBool, &out.DisableAlbumMetrics)
	base_config.OverlayFlag(flags, "disable-history-metrics", flags.GetBool, &out.DisableHistoryMetrics)
	base_config.OverlayFlag(flags, "disable-wanted-metrics", flags.GetBool, &out.DisableWantedMetrics)
	return out, nil
}

// Validate checks the configuration, including auth-credential pairing.
func (c *ArrConfig) Validate() error {
	var errs []error
	if c.URL == "" {
		errs = append(errs, errors.New("url is required"))
	} else if u, err := url.Parse(c.URL); err != nil || u.Scheme == "" || u.Host == "" {
		errs = append(errs, fmt.Errorf("url must be a valid URL: %q", c.URL))
	}
	if !apiKeyRegex.MatchString(c.APIKey) {
		errs = append(errs, errors.New("api-key must be a 20-32 character alphanumeric string"))
	}

	if c.FormAuth {
		if c.AuthUsername == "" || c.AuthPassword == "" {
			errs = append(errs, errors.New("auth-username and auth-password are required when form-auth is set"))
		}
	} else if c.AuthUsername != "" || c.AuthPassword != "" {
		errs = append(errs, errors.New("auth-username/auth-password are only supported with form-auth (basic auth was removed)"))
	}
	return errors.Join(errs...)
}
