package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type historyCollector struct {
	config        *config.Config   // App configuration
	historyMetric *prometheus.Desc // Total number of history items
	errorMetric   *prometheus.Desc // Error Description for use with InvalidMetric
}

func NewHistoryCollector(c *config.Config) *historyCollector {
	return &historyCollector{
		config: c,
		historyMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_history_total", c.Arr),
			"Total number of item in the history",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_history_collector_error", c.Arr),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
	}
}

func (collector *historyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.historyMetric
}

func (collector *historyCollector) Collect(ch chan<- prometheus.Metric) {
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorf("Error creating client: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	history := model.History{}
	if err := c.DoRequest("history", &history); err != nil {
		log.Errorf("Error getting history: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(collector.historyMetric, prometheus.GaugeValue, float64(history.TotalRecords))
}
