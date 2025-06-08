package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type queueCollector struct {
	config      *config.ArrConfig // App configuration
	queueMetric *prometheus.Desc  // Total number of queue items
	errorMetric *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewQueueCollector(c *config.ArrConfig) *queueCollector {
	return &queueCollector{
		config: c,
		queueMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_queue_total", c.App),
			"Total number of items in the queue by status, download_status, and download_state",
			[]string{"status", "download_status", "download_state"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_queue_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *queueCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.queueMetric
}

func (collector *queueCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "queue")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	params := client.QueryParams{}
	params.Add("page", "1")
	if collector.config.EnableUnknownQueueItems {
		switch collector.config.App {
		case "sonarr":
			params.Add("includeUnknownSeriesItems", "true")
		case "radarr":
			params.Add("includeUnknownMovieItems", "true")
		}
	}

	queue := model.Queue{}
	if err := c.DoRequest("queue", &queue, params); err != nil {
		log.Errorw("Error getting queue",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	// Calculate total pages
	var totalPages = (queue.TotalRecords + queue.PageSize - 1) / queue.PageSize
	// Paginate
	var queueStatusAll = make([]model.QueueRecords, 0, queue.TotalRecords)
	queueStatusAll = append(queueStatusAll, queue.Records...)
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			params.Set("page", fmt.Sprintf("%d", page))
			if err := c.DoRequest("queue", &queue, params); err != nil {
				log.Errorw("Error getting queue page",
					"page", page,
					"error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			queueStatusAll = append(queueStatusAll, queue.Records...)
		}
	}
	// Group metrics by status, download_status and download_state
	if len(queueStatusAll) > 0 {
		var queueMetrics prometheus.Metric
		for i, s := range queueStatusAll {
			queueMetrics = prometheus.MustNewConstMetric(collector.queueMetric, prometheus.GaugeValue, float64(i+1),
				s.Status, s.TrackedDownloadStatus, s.TrackedDownloadState,
			)
		}
		ch <- queueMetrics
	}
}
