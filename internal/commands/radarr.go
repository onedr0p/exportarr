package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	radarrCollector "github.com/onedr0p/exportarr/internal/collector/arr/radarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/arr/shared"
	"github.com/onedr0p/exportarr/internal/config"
)

func init() {
	rootCmd.AddCommand(radarrCmd)
	config.RegisterArrFlags(radarrCmd.PersistentFlags())
}

var radarrCmd = &cobra.Command{
	Use:     "radarr",
	Aliases: []string{"r"},
	Short:   "Prometheus Exporter for Radarr",
	Long:    "Prometheus Exporter for Radarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr.App = "radarr"
		conf.LoadArrFlags(cmd.PersistentFlags())
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
