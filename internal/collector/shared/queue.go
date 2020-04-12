package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type queueCollector struct {
	config      *cli.Context
	queueMetric *prometheus.Desc
}

func NewQueueCollector(c *cli.Context) *queueCollector {
	return &queueCollector{
		config: c,
		queueMetric: prometheus.NewDesc(
			"radarr_queue_total",
			"Total number of movies in the queue by status",
			[]string{"status", "download_status", "download_state"},
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *queueCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.queueMetric
}

func (collector *queueCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config)
	queue := model.Queue{}
	if err := c.DoRequest("queue?page=1&includeUnknownMovieItems=true", &queue); err != nil {
		log.Fatal(err)
	}
	// Calculate total pages
	var totalPages = (queue.TotalRecords + queue.PageSize - 1) / queue.PageSize
	// Paginate
	var queueStatusAll = make([]model.QueueRecords, 0, queue.TotalRecords)
	queueStatusAll = append(queueStatusAll, queue.Records...)
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			if err := client.DoRequest(fmt.Sprintf("%s?page=%d&includeUnknownMovieItems=true", "queue", page), &queue); err != nil {
				log.Fatal(err)
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
