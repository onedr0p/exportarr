package commands

import (
	"fmt"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/onedr0p/exportarr/internal/arr/collector"
	"github.com/onedr0p/exportarr/internal/arr/config"
)

func init() {
	config.RegisterArrFlags(radarrCmd.PersistentFlags())
	config.RegisterArrFlags(sonarrCmd.PersistentFlags())
	config.RegisterArrFlags(lidarrCmd.PersistentFlags())
	config.RegisterArrFlags(readarrCmd.PersistentFlags())
	config.RegisterArrFlags(prowlarrCmd.PersistentFlags())
	config.RegisterProwlarrFlags(prowlarrCmd.PersistentFlags())

	rootCmd.AddCommand(
		radarrCmd,
		sonarrCmd,
		lidarrCmd,
		readarrCmd,
		prowlarrCmd,
	)
}

func UsageOnError(cmd *cobra.Command, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		cmd.Usage()
		os.Exit(1)
	}
}

var radarrCmd = &cobra.Command{
	Use:     "radarr",
	Aliases: []string{"r"},
	Short:   "Prometheus Exporter for Radarr",
	Long:    "Prometheus Exporter for Radarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
		if err != nil {
			return err
		}
		c.ApiVersion = "v3"
		UsageOnError(cmd, c.Validate())

		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewRadarrCollector(c),
				collector.NewQueueCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewRootFolderCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c),
			)
		})
		return nil
	},
}

var sonarrCmd = &cobra.Command{
	Use:     "sonarr",
	Aliases: []string{"s"},
	Short:   "Prometheus Exporter for Sonarr",
	Long:    "Prometheus Exporter for Sonarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
		if err != nil {
			return err
		}
		c.ApiVersion = "v3"
		UsageOnError(cmd, c.Validate())

		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewSonarrCollector(c),
				collector.NewQueueCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewRootFolderCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c),
			)
		})
		return nil
	},
}

var lidarrCmd = &cobra.Command{
	Use:   "lidarr",
	Short: "Prometheus Exporter for Lidarr",
	Long:  "Prometheus Exporter for Lidarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
		if err != nil {
			return err
		}
		c.ApiVersion = "v1"
		UsageOnError(cmd, c.Validate())

		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewLidarrCollector(c),
				collector.NewQueueCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewRootFolderCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c),
			)
		})
		return nil
	},
}

var readarrCmd = &cobra.Command{
	Use:     "readarr",
	Aliases: []string{"b"},
	Short:   "Prometheus Exporter for Readarr",
	Long:    "Prometheus Exporter for Readarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
		if err != nil {
			return err
		}
		c.ApiVersion = "v1"
		UsageOnError(cmd, c.Validate())

		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewReadarrCollector(c),
				collector.NewQueueCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewRootFolderCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c),
			)
		})
		return nil
	},
}

var prowlarrCmd = &cobra.Command{
	Use:     "prowlarr",
	Aliases: []string{"p"},
	Short:   "Prometheus Exporter for Prowlarr",
	Long:    "Prometheus Exporter for Prowlarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.LoadArrConfig(*conf, cmd.PersistentFlags())
		if err != nil {
			return err
		}
		c.ApiVersion = "v1"
		c.LoadProwlarrConfig(cmd.PersistentFlags())
		if err := c.Prowlarr.Validate(); err != nil {
			return err
		}
		UsageOnError(cmd, c.Validate())
		UsageOnError(cmd, c.Prowlarr.Validate())

		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewProwlarrCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c,
					collector.NewUnavailableIndexerEmitter(c.URL)),
			)
		})
		return nil
	},
}
