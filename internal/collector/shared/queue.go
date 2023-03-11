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
	config      *cli.Context     // App configuration
	configFile  *model.Config    // *arr configuration from config.xml
	queueMetric *prometheus.Desc // Total number of queue items
}

func NewQueueCollector(c *cli.Context, cf *model.Config) *queueCollector {
	return &queueCollector{
		config:     c,
		configFile: cf,
		queueMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_queue_total", c.Command.Name),
			"Total number of items in the queue by status, download_status, and download_state",
			[]string{"status", "download_status", "download_state"},
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *queueCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.queueMetric
}

func (collector *queueCollector) Collect(ch chan<- prometheus.Metric) {
	c, err := client.NewClient(collector.config, collector.configFile)
	if err != nil {
		log.Errorf("Error creating client: %w", err)
		ch <- prometheus.NewInvalidMetric(
			prometheus.NewDesc(
				fmt.Sprintf("%s_collector_error", collector.config.Command.Name),
				"Error Collecting metrics",
				nil,
				prometheus.Labels{"url": collector.config.String("url")}),
			err)
		return
	}

	params := map[string]string{"page": "1"}
	if collector.config.Bool("enable-unknown-queue-items") {
		if collector.config.Command.Name == "sonarr" {
			params["includeUnknownSeriesItems"] = "true"
		} else if collector.config.Command.Name == "radarr" {
			params["includeUnknownMovieItems"] = "true"
		}
	}

	queue := model.Queue{}
	if err := c.DoRequest("queue", &queue, params); err != nil {
		log.Fatal(err)
	}
	// Calculate total pages
	var totalPages = (queue.TotalRecords + queue.PageSize - 1) / queue.PageSize
	// Paginate
	var queueStatusAll = make([]model.QueueRecords, 0, queue.TotalRecords)
	queueStatusAll = append(queueStatusAll, queue.Records...)
	if totalPages > 1 {
		for page := 2; page <= totalPages; page++ {
			params["page"] = fmt.Sprintf("%d", page)
			if err := c.DoRequest("queue", &queue, params); err != nil {
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
