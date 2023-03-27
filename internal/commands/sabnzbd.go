package commands

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(sabnzbdCmd)
}

var sabnzbdCmd = &cobra.Command{
	Use:     "sabnzbd",
	Aliases: []string{"sab"},
	Short:   "Prometheus Exporter for Sabnzbd",
	Long:    "Prometheus Exporter for Sabnzbd.",
	RunE: func(cmd *cobra.Command, args []string) error {
		serveHttp(func(r *prometheus.Registry) {
			r.MustRegister()
		})
		return nil
	},
}
