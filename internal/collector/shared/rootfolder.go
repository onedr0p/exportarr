package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type rootFolderCollector struct {
	config           *cli.Context
	rootFolderMetric *prometheus.Desc
}

func NewRootFolderCollector(c *cli.Context) *rootFolderCollector {
	return &rootFolderCollector{
		config: c,
		rootFolderMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_rootfolder_freespace_bytes", c.Command.Name),
			"Root folder space in bytes by path",
			[]string{"path"},
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *rootFolderCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.rootFolderMetric
}

func (collector *rootFolderCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config)
	rootFolders := model.RootFolder{}
	if err := c.DoRequest("rootfolder", &rootFolders); err != nil {
		log.Fatal(err)
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
