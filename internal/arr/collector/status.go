package collector

import (
	"log/slog"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type systemStatusCollector struct {
	client       *client.Client
	config       *config.ArrConfig // App configuration
	systemStatus *prometheus.Desc  // Total number of system statuses
	errorMetric  *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewSystemStatusCollector builds a collector for the system/status endpoint.
func NewSystemStatusCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &systemStatusCollector{
		client:       httpClient,
		config:       c,
		systemStatus: newDesc(c.App, "system_status", "System Status", nil, c.URL),
		errorMetric:  newDesc(c.App, "status_collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *systemStatusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.systemStatus
}

func (collector *systemStatusCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "system_status")
	defer recoverCollect(log, ch, collector.errorMetric)
	c := collector.client
	systemStatus, err := client.Get[model.SystemStatus](c, "system/status")
	if err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else if (model.SystemStatus{}) == systemStatus {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(0.0))
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatus, prometheus.GaugeValue, float64(1.0))
	}
}
