package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	readarrCollector "github.com/onedr0p/exportarr/internal/collector/arr/readarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/arr/shared"
	"github.com/onedr0p/exportarr/internal/config"
)

func init() {
	rootCmd.AddCommand(readarrCmd)
	config.RegisterArrFlags(readarrCmd.PersistentFlags())
}

var readarrCmd = &cobra.Command{
	Use:     "readarr",
	Aliases: []string{"b"},
	Short:   "Prometheus Exporter for Readarr",
	Long:    "Prometheus Exporter for Readarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr.App = "readarr"
		conf.Arr.ApiVersion = "v1"
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				readarrCollector.NewReadarrCollector(conf),
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
