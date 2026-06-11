package collector

import (
	"log/slog"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type rootFolderCollector struct {
	client           *client.Client
	config           *config.ArrConfig // App configuration
	rootFolderMetric *prometheus.Desc  // Total number of root folders
	errorMetric      *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewRootFolderCollector builds a collector for root-folder free space.
func NewRootFolderCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &rootFolderCollector{
		client:           httpClient,
		config:           c,
		rootFolderMetric: newDesc(c.App, "rootfolder_freespace_bytes", "Root folder space in bytes by path", []string{"path"}, c.URL),
		errorMetric:      newDesc(c.App, "rootfolder_collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *rootFolderCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.rootFolderMetric
}

func (collector *rootFolderCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "rootfolder")
	defer recoverCollect(log, ch, collector.errorMetric)
	c, err := client.NewClient(collector.config)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error creating client", "error", err)
		return
	}
	rootFolders, err := client.Get[model.RootFolder](c, "rootfolder")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting rootfolder", "error", err)
		return
	}
	// Group metrics by path
	if len(rootFolders) > 0 {
		for _, rootFolder := range rootFolders {
			ch <- prometheus.MustNewConstMetric(collector.rootFolderMetric, prometheus.GaugeValue, float64(rootFolder.FreeSpace),
				rootFolder.Path,
			)
		}
	}
}
