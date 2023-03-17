package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	radarrCollector "github.com/onedr0p/exportarr/internal/collector/radarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
)

func init() {
	rootCmd.AddCommand(radarrCmd)
}

var radarrCmd = &cobra.Command{
	Use:   "radarr",
	Short: "Prometheus Exporter for Radarr",
	Long:  "Prometheus Exporter for Radarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr = "radarr"
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				radarrCollector.NewRadarrCollector(conf),
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
