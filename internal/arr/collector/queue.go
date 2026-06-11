package collector

import (
	"log/slog"
	"strconv"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type queueCollector struct {
	client      *client.Client
	config      *config.ArrConfig // App configuration
	queueMetric *prometheus.Desc  // Total number of queue items
	errorMetric *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewQueueCollector builds a collector for the *arr queue endpoint.
func NewQueueCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &queueCollector{
		client:      httpClient,
		config:      c,
		queueMetric: newDesc(c.App, "queue_total", "Total number of items in the queue by status, download_status, and download_state", []string{"status", "download_status", "download_state"}, c.URL),
		errorMetric: newDesc(c.App, "queue_collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *queueCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.queueMetric
}

func (collector *queueCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "queue")
	defer recoverCollect(log, ch, collector.errorMetric)
	c, err := client.NewClient(collector.config)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error creating client", "error", err)
		return
	}

	params := client.QueryParams{}
	params.Add("page", "1")
	params.Add("pageSize", "250")
	if collector.config.EnableUnknownQueueItems {
		switch collector.config.App {
		case "sonarr":
			params.Add("includeUnknownSeriesItems", "true")
		case "radarr":
			params.Add("includeUnknownMovieItems", "true")
		}
	}

	queue, err := client.Get[model.Queue](c, "queue", params)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting queue", "error", err)
		return
	}
	// Calculate total pages, guarding against a zero page size in the response.
	totalPages := 0
	if queue.PageSize > 0 {
		totalPages = (queue.TotalRecords + queue.PageSize - 1) / queue.PageSize
	}
	// Paginate
	queueStatusAll := make([]model.QueueRecords, 0, queue.TotalRecords)
	queueStatusAll = append(queueStatusAll, queue.Records...)
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			params.Set("page", strconv.Itoa(page))
			queue, err = client.Get[model.Queue](c, "queue", params)
			if err != nil {
				emitError(log, ch, collector.errorMetric, "Error getting queue page", "page", page, "error", err)
				return
			}
			queueStatusAll = append(queueStatusAll, queue.Records...)
		}
	}
	// Group metrics by status, download_status and download_state, one series
	// per distinct label combination.
	counts := map[[3]string]int{}
	for _, s := range queueStatusAll {
		counts[[3]string{s.Status, s.TrackedDownloadStatus, s.TrackedDownloadState}]++
	}
	// An empty queue must still produce a series so dashboards can tell
	// "zero items" apart from "scrape failed" (#389).
	if len(counts) == 0 {
		ch <- prometheus.MustNewConstMetric(collector.queueMetric, prometheus.GaugeValue, 0, "", "", "")
		return
	}
	for labels, count := range counts {
		ch <- prometheus.MustNewConstMetric(collector.queueMetric, prometheus.GaugeValue, float64(count),
			labels[0], labels[1], labels[2],
		)
	}
}
