package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type systemHealthCollector struct {
	config             *cli.Context
	systemHealthMetric *prometheus.Desc
}

func NewSystemHealthCollector(c *cli.Context) *systemHealthCollector {
	return &systemHealthCollector{
		config: c,
		systemHealthMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_system_health_issues", c.Command.Name),
			"Total number of health issues by source, type, message and wikiurl",
			[]string{"source", "type", "message", "wikiurl"},
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *systemHealthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.systemHealthMetric
}

func (collector *systemHealthCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config)
	systemHealth := model.SystemHealth{}
	if err := c.DoRequest("health", &systemHealth); err != nil {
		log.Fatal(err)
	}
	// Group metrics by source, type, message and wikiurl
	if len(systemHealth) > 0 {
		for _, s := range systemHealth {
			ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(1),
				s.Source, s.Type, s.Message, s.WikiURL,
			)
		}
	}
}
