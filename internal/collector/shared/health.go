package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type systemHealthCollector struct {
	config             *config.Config   // App configuration
	systemHealthMetric *prometheus.Desc // Total number of health issues
	errorMetric        *prometheus.Desc // Error Description for use with InvalidMetric
}

func NewSystemHealthCollector(c *config.Config) *systemHealthCollector {
	return &systemHealthCollector{
		config: c,
		systemHealthMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_system_health_issues", c.Arr),
			"Total number of health issues by source, type, message and wikiurl",
			[]string{"source", "type", "message", "wikiurl"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_health_collector_error", c.Arr),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
	}
}

func (collector *systemHealthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.systemHealthMetric
}

func (collector *systemHealthCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "systemHealth")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorf("Error creating client: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	systemHealth := model.SystemHealth{}
	if err := c.DoRequest("health", &systemHealth); err != nil {
		log.Errorf("Error getting health: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	// Group metrics by source, type, message and wikiurl
	if len(systemHealth) > 0 {
		for _, s := range systemHealth {
			ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(1),
				s.Source, s.Type, s.Message, s.WikiURL,
			)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(0), "", "", "", "")
	}
}
