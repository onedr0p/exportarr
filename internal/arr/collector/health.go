package collector

import (
	"log/slog"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type systemHealthCollector struct {
	client             *client.Client
	config             *config.ArrConfig          // App configuration
	systemHealthMetric *prometheus.Desc           // Total number of health issues
	errorMetric        *prometheus.Desc           // Error Description for use with InvalidMetric
	extraEmitters      []ExtraHealthMetricEmitter // Registered Emitters for extra per-app metrics
}

// ExtraHealthMetricEmitter emits app-specific metrics derived from health
// messages.
type ExtraHealthMetricEmitter interface {
	Describe() *prometheus.Desc
	Emit(model.SystemHealthMessage) []prometheus.Metric
}

// NewSystemHealthCollector builds a collector for the system/health endpoint.
func NewSystemHealthCollector(httpClient *client.Client, c *config.ArrConfig, emitters ...ExtraHealthMetricEmitter) prometheus.Collector {
	return &systemHealthCollector{
		client:             httpClient,
		config:             c,
		systemHealthMetric: newDesc(c.App, "system_health_issues", "Total number of health issues by source, type, message and wikiurl", []string{"source", "type", "message", "wikiurl"}, c.URL),
		errorMetric:        newDesc(c.App, "health_collector_error", "Error while collecting metrics", nil, c.URL),
		extraEmitters:      emitters,
	}
}

func (collector *systemHealthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.systemHealthMetric
	for _, emitter := range collector.extraEmitters {
		ch <- emitter.Describe()
	}
}

func (collector *systemHealthCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "systemHealth")
	defer recoverCollect(log, ch, collector.errorMetric)
	c := collector.client
	systemHealth, err := client.Get[model.SystemHealth](c, "health")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting health", "error", err)
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
