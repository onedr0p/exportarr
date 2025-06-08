package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type systemStatusCollector struct {
	config       *config.ArrConfig // App configuration
	systemStatus *prometheus.Desc  // Total number of system statuses
	errorMetric  *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewSystemStatusCollector(c *config.ArrConfig) *systemStatusCollector {
	return &systemStatusCollector{
		config: c,
		systemStatus: prometheus.NewDesc(
			fmt.Sprintf("%s_system_status", c.App),
			"System Status",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_status_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *systemStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.systemStatus
}

func (collector *systemStatusCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "system_status")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	systemStatus := model.SystemStatus{}
	if err := c.DoRequest("system/status", &systemStatus); err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else if (model.SystemStatus{}) == systemStatus {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(1.0))
	}
}
