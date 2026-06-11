package collector

import (
	"log/slog"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type historyCollector struct {
	client        *client.Client
	config        *config.ArrConfig // App configuration
	historyMetric *prometheus.Desc  // Total number of history items
	errorMetric   *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewHistoryCollector builds a collector for the *arr history endpoint.
func NewHistoryCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &historyCollector{
		client:        httpClient,
		config:        c,
		historyMetric: newDesc(c.App, "history_total", "Total number of item in the history", nil, c.URL),
		errorMetric:   newDesc(c.App, "history_collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *historyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.historyMetric
}

func (collector *historyCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "history")
	defer recoverCollect(log, ch, collector.errorMetric)
	c, err := client.NewClient(collector.config)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error creating client", "error", err)
		return
	}
	// Only totalRecords is read: request the smallest page the API allows.
	params := client.QueryParams{}
	params.Add("pageSize", "1")
	history, err := client.Get[model.History](c, "history", params)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting history", "error", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(collector.historyMetric, prometheus.GaugeValue, float64(history.TotalRecords))
}
