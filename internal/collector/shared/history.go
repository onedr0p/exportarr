package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type historyCollector struct {
	config        *cli.Context     // App configuration
	configFile    *model.Config    // *arr configuration from config.xml
	historyMetric *prometheus.Desc // Total number of history items
}

func NewHistoryCollector(c *cli.Context, cf *model.Config) *historyCollector {
	return &historyCollector{
		config:     c,
		configFile: cf,
		historyMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_history_total", c.Command.Name),
			"Total number of item in the history",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *historyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.historyMetric
}

func (collector *historyCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config, collector.configFile)
	history := model.History{}
	if err := c.DoRequest("history", &history); err != nil {
		log.Fatal(err)
	}
	ch <- prometheus.MustNewConstMetric(collector.historyMetric, prometheus.GaugeValue, float64(history.TotalRecords))
}
