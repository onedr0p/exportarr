package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
	sonarrCollector "github.com/onedr0p/exportarr/internal/collector/sonarr"
)

func init() {
	rootCmd.AddCommand(sonarrCmd)
}

var sonarrCmd = &cobra.Command{
	Use:     "sonarr",
	Aliases: []string{"s"},
	Short:   "Prometheus Exporter for Sonarr",
	Long:    "Prometheus Exporter for Sonarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr = "sonarr"
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
