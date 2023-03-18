package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	lidarrCollector "github.com/onedr0p/exportarr/internal/collector/lidarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
)

func init() {
	rootCmd.AddCommand(lidarrCmd)
}

var lidarrCmd = &cobra.Command{
	Use:   "lidarr",
	Short: "Prometheus Exporter for Lidarr",
	Long:  "Prometheus Exporter for Lidarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr = "lidarr"
		conf.ApiVersion = "v1"
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				lidarrCollector.NewLidarrCollector(conf),
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
