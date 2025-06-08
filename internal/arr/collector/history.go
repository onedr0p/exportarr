package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type historyCollector struct {
	config        *config.ArrConfig // App configuration
	historyMetric *prometheus.Desc  // Total number of history items
	errorMetric   *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewHistoryCollector(c *config.ArrConfig) *historyCollector {
	return &historyCollector{
		config: c,
		historyMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_history_total", c.App),
			"Total number of item in the history",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_history_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *historyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.historyMetric
}

func (collector *historyCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "history")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	history := model.History{}
	if err := c.DoRequest("history", &history); err != nil {
		log.Errorw("Error getting history",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(collector.historyMetric, prometheus.GaugeValue, float64(history.TotalRecords))
}
