package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type rootFolderCollector struct {
	config           *config.ArrConfig // App configuration
	rootFolderMetric *prometheus.Desc  // Total number of root folders
	errorMetric      *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewRootFolderCollector(c *config.ArrConfig) *rootFolderCollector {
	return &rootFolderCollector{
		config: c,
		rootFolderMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_rootfolder_freespace_bytes", c.App),
			"Root folder space in bytes by path",
			[]string{"path"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_rootfolder_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *rootFolderCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.rootFolderMetric
}

func (collector *rootFolderCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "rootfolder")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	rootFolders := model.RootFolder{}
	if err := c.DoRequest("rootfolder", &rootFolders); err != nil {
		log.Errorw("Error getting rootfolder",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
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
