package collector

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/onedr0p/exportarr/internal/assert"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

// panickyCollector simulates a collector hitting an unexpected payload.
type panickyCollector struct {
	errorMetric *prometheus.Desc
}

func (p *panickyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.errorMetric
}

func (p *panickyCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "panicky")
	defer recoverCollect(log, ch, p.errorMetric)
	panic("unexpected payload")
}

// TestRecoverCollect proves a panicking collector degrades to its error gauge
// instead of crashing the process: prometheus gathers collectors in goroutines
// that no HTTP middleware can recover for us.
func TestRecoverCollect(t *testing.T) {
	c := &panickyCollector{
		errorMetric: newDesc("panicky", "collector_error", "Error while collecting metrics", nil, "http://x"),
	}
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(c)

	families, err := registry.Gather()
	assert.NoError(t, err, "a collector panic must not fail the gather")
	assert.Equal(t, len(families), 1)
	assert.Equal(t, families[0].GetName(), "panicky_collector_error")
}

// panickyWorkerCollector simulates a fan-out worker (errgroup goroutine)
// hitting an unexpected payload — those goroutines are outside Collect's
// recover, so goRecoverable must convert the panic into a Wait error.
type panickyWorkerCollector struct {
	errorMetric *prometheus.Desc
}

func (p *panickyWorkerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- p.errorMetric
}

func (p *panickyWorkerCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "panicky_worker")
	defer recoverCollect(log, ch, p.errorMetric)
	eg := errgroup.Group{}
	goRecoverable(&eg, func() error {
		panic("unexpected worker payload")
	})
	if err := eg.Wait(); err != nil {
		ch <- prometheus.MustNewConstMetric(p.errorMetric, prometheus.GaugeValue, 1)
		return
	}
}

// TestGoRecoverable proves a panic inside an errgroup worker degrades to the
// collector's error gauge — errgroup itself does not recover worker panics,
// and an unrecovered worker panic would crash the whole exporter.
func TestGoRecoverable(t *testing.T) {
	eg := errgroup.Group{}
	goRecoverable(&eg, func() error { panic("boom") })
	err := eg.Wait()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "worker panicked"), "got: %v", err)

	c := &panickyWorkerCollector{
		errorMetric: newDesc("panickyworker", "collector_error", "Error while collecting metrics", nil, "http://x"),
	}
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(c)

	families, err := registry.Gather()
	assert.NoError(t, err, "a worker panic must not fail the gather")
	assert.Equal(t, len(families), 1)
	assert.Equal(t, families[0].GetName(), "panickyworker_collector_error")
}
