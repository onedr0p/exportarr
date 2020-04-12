package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli/v2"
)

type systemStatusCollector struct {
	config       *cli.Context
	systemStatus *prometheus.Desc
}

func NewSystemStatusCollector(c *cli.Context) *systemStatusCollector {
	return &systemStatusCollector{
		config: c,
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
	c := client.NewClient(collector.config)
	systemStatus := model.SystemStatus{}
	if err := c.DoRequest("system/status", &systemStatus); err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else if (model.SystemStatus{}) == systemStatus {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(1.0))
	}
}
