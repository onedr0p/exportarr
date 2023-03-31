package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	"github.com/onedr0p/exportarr/internal/arr/collector"
	"github.com/onedr0p/exportarr/internal/arr/config"
)

func init() {
	rootCmd.AddCommand(radarrCmd)
	config.RegisterArrFlags(radarrCmd.PersistentFlags())

	rootCmd.AddCommand(sonarrCmd)
	config.RegisterArrFlags(sonarrCmd.PersistentFlags())

	rootCmd.AddCommand(lidarrCmd)
	config.RegisterArrFlags(lidarrCmd.PersistentFlags())

	rootCmd.AddCommand(readarrCmd)
	config.RegisterArrFlags(readarrCmd.PersistentFlags())

	rootCmd.AddCommand(prowlarrCmd)
	config.RegisterArrFlags(prowlarrCmd.PersistentFlags())
	prowlarrCmd.PersistentFlags().Bool("backfill", false, "Backfill Prowlarr")
	prowlarrCmd.PersistentFlags().String("backfill-since-date", "", "Date from which to start Prowlarr Backfill")
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
		c.App = "radarr"
		c.ApiVersion = "v3"
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
		c.App = "sonarr"
		c.ApiVersion = "v3"
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
		c.App = "lidarr"
		c.ApiVersion = "v1"
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
		c.App = "readarr"
		c.ApiVersion = "v1"
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
		c.App = "prowlarr"
		c.ApiVersion = "v1"
		c.LoadProwlarrFlags(cmd.PersistentFlags())
		if err := c.Prowlarr.Validate(); err != nil {
			return err
		}
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				collector.NewProwlarrCollector(c),
				collector.NewHistoryCollector(c),
				collector.NewSystemStatusCollector(c),
				collector.NewSystemHealthCollector(c),
			)
		})
		return nil
	},
}
