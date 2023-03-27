package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	sharedCollector "github.com/onedr0p/exportarr/internal/collector/arr/shared"
	sonarrCollector "github.com/onedr0p/exportarr/internal/collector/arr/sonarr"
	"github.com/onedr0p/exportarr/internal/config"
)

func init() {
	rootCmd.AddCommand(sonarrCmd)
	config.RegisterArrFlags(sonarrCmd.PersistentFlags())
}

var sonarrCmd = &cobra.Command{
	Use:     "sonarr",
	Aliases: []string{"s"},
	Short:   "Prometheus Exporter for Sonarr",
	Long:    "Prometheus Exporter for Sonarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr.App = "sonarr"
		conf.LoadArrFlags(cmd.PersistentFlags())
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				sonarrCollector.NewSonarrCollector(conf),
				sharedCollector.NewQueueCollector(conf),
				sharedCollector.NewHistoryCollector(conf),
				sharedCollector.NewRootFolderCollector(conf),
				sharedCollector.NewSystemStatusCollector(conf),
				sharedCollector.NewSystemHealthCollector(conf),
			)
		})
		return nil
	},
}
