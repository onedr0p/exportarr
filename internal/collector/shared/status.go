package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

type systemStatusCollector struct {
	config       *cli.Context     // App configuration
	configFile   *model.Config    // *arr configuration from config.xml
	systemStatus *prometheus.Desc // Total number of system statuses
}

func NewSystemStatusCollector(c *cli.Context, cf *model.Config) *systemStatusCollector {
	return &systemStatusCollector{
		config:     c,
		configFile: cf,
		systemStatus: prometheus.NewDesc(
			fmt.Sprintf("%s_system_status", c.Command.Name),
			"System Status",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *systemStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.systemStatus
}

func (collector *systemStatusCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config, collector.configFile)
	systemStatus := model.SystemStatus{}
	if err := c.DoRequest("system/status", &systemStatus); err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else if (model.SystemStatus{}) == systemStatus {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(1.0))
	}
}
