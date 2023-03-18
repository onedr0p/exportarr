package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"

	prowlarrCollector "github.com/onedr0p/exportarr/internal/collector/prowlarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
)

func init() {
	rootCmd.AddCommand(prowlarrCmd)

	prowlarrCmd.PersistentFlags().Bool("backfill", false, "Backfill Prowlarr")
	prowlarrCmd.PersistentFlags().String("backfill-since-date", "", "Date from which to start Prowlarr Backfill")
}

var prowlarrCmd = &cobra.Command{
	Use:     "prowlarr",
	Aliases: []string{"p"},
	Short:   "Prometheus Exporter for Prowlarr",
	Long:    "Prometheus Exporter for Prowlarr.",
	RunE: func(cmd *cobra.Command, args []string) error {
		conf.Arr = "prowlarr"
		conf.ApiVersion = "v1"
		conf.LoadProwlarrFlags(cmd.PersistentFlags())
		if err := conf.Prowlarr.Validate(); err != nil {
			return err
		}
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister(
				prowlarrCollector.NewProwlarrCollector(conf),
				sharedCollector.NewHistoryCollector(conf),
				sharedCollector.NewSystemStatusCollector(conf),
				sharedCollector.NewSystemHealthCollector(conf),
			)
		})
		return nil
	},
}
