// Package commands wires the exportarr CLI: one subcommand per supported app.
package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/collector"
	"github.com/onedr0p/exportarr/internal/arr/config"
)

func init() {
	config.RegisterArrFlags(radarrCmd.PersistentFlags())
	config.RegisterArrFlags(sonarrCmd.PersistentFlags())
	config.RegisterArrFlags(lidarrCmd.PersistentFlags())
	config.RegisterArrFlags(prowlarrCmd.PersistentFlags())
	config.RegisterArrFlags(bazarrCmd.PersistentFlags())
	config.RegisterProwlarrFlags(prowlarrCmd.PersistentFlags())
	config.RegisterBazarrFlags(bazarrCmd.PersistentFlags())

	rootCmd.AddCommand(
		radarrCmd,
		sonarrCmd,
		lidarrCmd,
		bazarrCmd,
		prowlarrCmd,
	)
}

// arrCommand describes what varies between the *arr subcommands; runE supplies
// the shared skeleton (load config → validate → build client → serve).
type arrCommand struct {
	apiVersion string
	// loadExtra parses app-specific flags/env into the sub-config (optional).
	loadExtra func(*config.ArrConfig, *flag.FlagSet) error
	// validateExtra checks the sub-config (optional).
	validateExtra func(*config.ArrConfig) error
	// collectors builds the collectors to register for this app.
	collectors func(*client.Client, *config.ArrConfig) []prometheus.Collector
}

func (a arrCommand) runE(cmd *cobra.Command, _ []string) error {
	c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
	if err != nil {
		return err
	}
	c.APIVersion = a.apiVersion
	if a.loadExtra != nil {
		if err := a.loadExtra(c, cmd.PersistentFlags()); err != nil {
			return err
		}
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if a.validateExtra != nil {
		if err := a.validateExtra(c); err != nil {
			return err
		}
	}
	httpClient, err := client.NewClient(c)
	if err != nil {
		return err
	}
	return serveHTTP(func(r prometheus.Registerer) {
		r.MustRegister(a.collectors(httpClient, c)...)
	})
}

// sharedArrCollectors returns the collectors common to the full *arr apps
// (radarr, sonarr, lidarr): queue, root folder, disk space, status, health,
// and — unless disabled — history.
func sharedArrCollectors(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
	out := []prometheus.Collector{
		collector.NewQueueCollector(httpClient, c),
		collector.NewRootFolderCollector(httpClient, c),
		collector.NewDiskSpaceCollector(httpClient, c),
		collector.NewSystemStatusCollector(httpClient, c),
		collector.NewSystemHealthCollector(httpClient, c),
	}
	if !c.DisableHistoryMetrics {
		out = append(out, collector.NewHistoryCollector(httpClient, c))
	}
	return out
}

var radarrCmd = &cobra.Command{
	Use:     "radarr",
	Aliases: []string{"r"},
	Short:   "Prometheus Exporter for Radarr",
	Long:    "Prometheus Exporter for Radarr.",
	RunE: arrCommand{
		apiVersion: "v3",
		collectors: func(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
			return append(sharedArrCollectors(httpClient, c), collector.NewRadarrCollector(httpClient, c))
		},
	}.runE,
}

var sonarrCmd = &cobra.Command{
	Use:     "sonarr",
	Aliases: []string{"s"},
	Short:   "Prometheus Exporter for Sonarr",
	Long:    "Prometheus Exporter for Sonarr.",
	RunE: arrCommand{
		apiVersion: "v3",
		collectors: func(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
			return append(sharedArrCollectors(httpClient, c), collector.NewSonarrCollector(httpClient, c))
		},
	}.runE,
}

var lidarrCmd = &cobra.Command{
	Use:   "lidarr",
	Short: "Prometheus Exporter for Lidarr",
	Long:  "Prometheus Exporter for Lidarr.",
	RunE: arrCommand{
		apiVersion: "v1",
		collectors: func(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
			return append(sharedArrCollectors(httpClient, c), collector.NewLidarrCollector(httpClient, c))
		},
	}.runE,
}

var bazarrCmd = &cobra.Command{
	Use:     "bazarr",
	Aliases: []string{"b"},
	Short:   "Prometheus Exporter for Bazarr",
	Long:    "Prometheus Exporter for Bazarr.",
	RunE: arrCommand{
		apiVersion: "",
		loadExtra: func(c *config.ArrConfig, flags *flag.FlagSet) error {
			return c.LoadBazarrConfig(flags)
		},
		validateExtra: func(c *config.ArrConfig) error { return c.Bazarr.Validate() },
		collectors: func(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
			return []prometheus.Collector{collector.NewBazarrCollector(httpClient, c)}
		},
	}.runE,
}

var prowlarrCmd = &cobra.Command{
	Use:     "prowlarr",
	Aliases: []string{"p"},
	Short:   "Prometheus Exporter for Prowlarr",
	Long:    "Prometheus Exporter for Prowlarr.",
	RunE: arrCommand{
		apiVersion: "v1",
		loadExtra: func(c *config.ArrConfig, flags *flag.FlagSet) error {
			return c.LoadProwlarrConfig(flags)
		},
		validateExtra: func(c *config.ArrConfig) error { return c.Prowlarr.Validate() },
		collectors: func(httpClient *client.Client, c *config.ArrConfig) []prometheus.Collector {
			out := []prometheus.Collector{
				collector.NewProwlarrCollector(httpClient, c),
				collector.NewSystemStatusCollector(httpClient, c),
				collector.NewSystemHealthCollector(httpClient, c,
					collector.NewUnavailableIndexerEmitter(c.URL)),
			}
			if !c.DisableHistoryMetrics {
				out = append(out, collector.NewHistoryCollector(httpClient, c))
			}
			return out
		},
	}.runE,
}
