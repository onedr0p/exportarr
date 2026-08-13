package commands

import (
	"errors"
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	arrconfig "github.com/onedr0p/exportarr/internal/arr/config"
	base_config "github.com/onedr0p/exportarr/internal/config"
	sabcollector "github.com/onedr0p/exportarr/internal/sabnzbd/collector"
	sabconfig "github.com/onedr0p/exportarr/internal/sabnzbd/config"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve metrics for every configured app from one process",
	Long: `Serve metrics for every configured app from a single exportarr process.

An app is enabled by setting its <APP>__URL and <APP>__API_KEY environment
variables, e.g. RADARR__URL/RADARR__API_KEY, SONARR__URL/SONARR__API_KEY,
LIDARR__URL/LIDARR__API_KEY, BAZARR__URL/BAZARR__API_KEY,
PROWLARR__URL/PROWLARR__API_KEY, SABNZBD__URL/SABNZBD__API_KEY. Every other
per-app env var (AUTH_USERNAME, FORM_AUTH, DISABLE_SSL_VERIFY,
REQUEST_TIMEOUT, ...) takes the same prefix. Each enabled app is served at
/metrics/<app>; scrape-tuning flags (DISABLE_QUALITY_METRICS, PROWLARR__BACKFILL,
...) apply process-wide, same as running the single-app subcommands.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		targets, err := buildMultiTargets()
		if err != nil {
			return err
		}
		return serveHTTP(targets)
	},
}

// multiArrApps enumerates the *arr apps eligible for multi-app mode: each
// entry's env-var prefix and the arrCommand that already knows how to build
// that app's collectors — shared with the single-app subcommands so both
// paths wire up a given app identically.
var multiArrApps = []struct {
	name   string
	prefix string
	app    arrCommand
}{
	{"radarr", "RADARR__", radarrApp},
	{"sonarr", "SONARR__", sonarrApp},
	{"lidarr", "LIDARR__", lidarrApp},
	{"bazarr", "BAZARR__", bazarrApp},
	{"prowlarr", "PROWLARR__", prowlarrApp},
}

// buildMultiTargets inspects the environment for every supported app and
// returns one metricsTarget per configured instance, mounted at
// /metrics/<app>. An app is configured iff its <PREFIX>URL is set.
func buildMultiTargets() ([]metricsTarget, error) {
	// Multi-app mode is env-only (no CLI flags), so every OverlayFlag call
	// below sees an empty, always-unchanged flag set and is a no-op —
	// LoadArrConfig/LoadBazarrConfig/LoadProwlarrConfig still run their
	// derived-field logic (e.g. Prowlarr's BackfillSinceTime parsing).
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)

	// Process-wide *arr settings (DISABLE_QUALITY_METRICS, BAZARR__...,
	// PROWLARR__..., etc.) parsed once and shared by every *arr instance.
	shared, err := arrconfig.LoadArrConfig(base_config.Config{}, flags)
	if err != nil {
		return nil, err
	}

	var targets []metricsTarget
	var errs []error

	for _, spec := range multiArrApps {
		inst, err := arrconfig.LoadArrInstanceEnv(spec.prefix)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
			continue
		}
		if !inst.Configured() {
			continue
		}

		c := arrconfig.NewArrConfigForInstance(*shared, spec.name, spec.app.apiVersion, *inst)
		if spec.app.loadExtra != nil {
			if err := spec.app.loadExtra(c, flags); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
				continue
			}
		}
		if err := c.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
			continue
		}
		if spec.app.validateExtra != nil {
			if err := spec.app.validateExtra(c); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
				continue
			}
		}
		cs, err := spec.app.buildCollectors(c)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", spec.name, err))
			continue
		}
		targets = append(targets, metricsTarget{
			path: spec.name,
			app:  c.App,
			url:  c.URL,
			register: func(r prometheus.Registerer) {
				r.MustRegister(cs...)
			},
		})
	}

	if sabTarget, err, ok := buildSabnzbdTarget(); err != nil {
		errs = append(errs, err)
	} else if ok {
		targets = append(targets, sabTarget)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	if len(targets) == 0 {
		return nil, errors.New(
			"no app instances configured for `serve`; set at least one of " +
				"RADARR__URL, SONARR__URL, LIDARR__URL, BAZARR__URL, PROWLARR__URL, " +
				"SABNZBD__URL (with the matching _API_KEY)")
	}
	return targets, nil
}

// buildSabnzbdTarget builds the SABnzbd metricsTarget if SABNZBD__URL is
// set. ok is false when SABnzbd is simply not configured (not an error).
func buildSabnzbdTarget() (target metricsTarget, err error, ok bool) {
	inst, err := sabconfig.LoadSabnzbdInstanceEnv("SABNZBD__")
	if err != nil {
		return metricsTarget{}, fmt.Errorf("sabnzbd: %w", err), false
	}
	if !inst.Configured() {
		return metricsTarget{}, nil, false
	}

	c := sabconfig.NewSabnzbdConfigForInstance(*inst)
	if err := c.Validate(); err != nil {
		return metricsTarget{}, fmt.Errorf("sabnzbd: %w", err), false
	}
	col, err := sabcollector.NewSabnzbdCollector(c)
	if err != nil {
		return metricsTarget{}, fmt.Errorf("sabnzbd: %w", err), false
	}
	return metricsTarget{
		path: "sabnzbd",
		app:  "sabnzbd",
		url:  c.URL,
		register: func(r prometheus.Registerer) {
			r.MustRegister(col)
		},
	}, nil, true
}
