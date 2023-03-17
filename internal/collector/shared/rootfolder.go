package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

type rootFolderCollector struct {
	config           *config.Config   // App configuration
	rootFolderMetric *prometheus.Desc // Total number of root folders
	errorMetric      *prometheus.Desc // Error Description for use with InvalidMetric
}

func NewRootFolderCollector(c *config.Config) *rootFolderCollector {
	return &rootFolderCollector{
		config: c,
		rootFolderMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_rootfolder_freespace_bytes", c.Arr),
			"Root folder space in bytes by path",
			[]string{"path"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_rootfolder_collector_error", c.Arr),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
	}
}

func (collector *rootFolderCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.rootFolderMetric
}

func (collector *rootFolderCollector) Collect(ch chan<- prometheus.Metric) {
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorf("Error creating client: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	rootFolders := model.RootFolder{}
	if err := c.DoRequest("rootfolder", &rootFolders); err != nil {
		log.Errorf("Error getting rootfolder: %s", err)
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
