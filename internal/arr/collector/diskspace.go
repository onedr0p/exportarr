package collector

import (
	"log/slog"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type diskSpaceCollector struct {
	client           *client.Client
	config           *config.ArrConfig // App configuration
	freeBytesMetric  *prometheus.Desc  // Free bytes per disk
	totalBytesMetric *prometheus.Desc  // Total bytes per disk
	errorMetric      *prometheus.Desc  // Error gauge for use on collection failure
}

// NewDiskSpaceCollector builds a collector for the shared diskspace endpoint,
// exposing free and total bytes per disk so usage can be computed.
func NewDiskSpaceCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &diskSpaceCollector{
		client: httpClient,
		config: c,
		freeBytesMetric: newDesc(c.App, "diskspace_free_bytes",
			"Free disk space in bytes by path and label", []string{"path", "label"}, c.URL),
		totalBytesMetric: newDesc(c.App, "diskspace_total_bytes",
			"Total disk space in bytes by path and label", []string{"path", "label"}, c.URL),
		errorMetric: newDesc(c.App, "diskspace_collector_error",
			"Error while collecting metrics", nil, c.URL),
	}
}

// Describe implements prometheus.Collector.
func (collector *diskSpaceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.freeBytesMetric
	ch <- collector.totalBytesMetric
}

// Collect implements prometheus.Collector.
func (collector *diskSpaceCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "diskspace")
	defer recoverCollect(log, ch, collector.errorMetric)
	c := collector.client

	disks, err := client.Get[model.DiskSpace](c, "diskspace")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting diskspace", "error", err)
		return
	}
	for _, disk := range disks {
		ch <- prometheus.MustNewConstMetric(collector.freeBytesMetric, prometheus.GaugeValue, float64(disk.FreeSpace),
			disk.Path, disk.Label,
		)
		ch <- prometheus.MustNewConstMetric(collector.totalBytesMetric, prometheus.GaugeValue, float64(disk.TotalSpace),
			disk.Path, disk.Label,
		)
	}
}
