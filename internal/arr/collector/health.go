package collector

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type systemHealthCollector struct {
	config             *config.ArrConfig          // App configuration
	systemHealthMetric *prometheus.Desc           // Total number of health issues
	errorMetric        *prometheus.Desc           // Error Description for use with InvalidMetric
	extraEmitters      []ExtraHealthMetricEmitter // Registered Emitters for extra per-app metrics
}

type ExtraHealthMetricEmitter interface {
	Describe() *prometheus.Desc
	Emit(model.SystemHealthMessage) []prometheus.Metric
}

func NewSystemHealthCollector(c *config.ArrConfig, emitters ...ExtraHealthMetricEmitter) *systemHealthCollector {
	return &systemHealthCollector{
		config: c,
		systemHealthMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_system_health_issues", c.App),
			"Total number of health issues by source, type, message and wikiurl",
			[]string{"source", "type", "message", "wikiurl"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_health_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		extraEmitters: emitters,
	}
}

func (collector *systemHealthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.systemHealthMetric
	for _, emitter := range collector.extraEmitters {
		ch <- emitter.Describe()
	}
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
			for _, emitter := range collector.extraEmitters {
				for _, metric := range emitter.Emit(s) {
					ch <- metric
				}
			}
		}
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(0), "", "", "", "")
	}
}
