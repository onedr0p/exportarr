package collector

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/shamelin/exportarr/internal/arr/model"
	"github.com/stretchr/testify/require"
)

type testCollector struct {
	emitter ExtraHealthMetricEmitter
	msg     model.SystemHealthMessage
}

func (c *testCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.emitter.Describe()
}

func (c *testCollector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range c.emitter.Emit(c.msg) {
		ch <- metric
	}
}

func TestUnavailableIndexerEmitter(t *testing.T) {
	emitter := UnavailableIndexerEmitter{
		url: "http://localhost:9117",
	}

	require := require.New(t)
	require.NotNil(emitter.Describe())

	msg := model.SystemHealthMessage{
		Source:  "IndexerLongTermStatusCheck",
		Type:    "warning",
		WikiURL: "https://wiki.servarr.com/prowlarr/system#indexers-are-unavailable-due-to-failures",
		Message: "Indexers unavailable due to failures for more than 6 hours: Server1, ServerTwo, ServerTHREE, Server.four",
	}
	metrics := emitter.Emit(msg)
	require.Len(metrics, 4)

	testCol := &testCollector{
		emitter: &emitter,
		msg:     msg,
	}

	expected := strings.NewReader(
		`# HELP prowlarr_indexer_unavailable Indexers marked unavailable due to repeated errors
		# TYPE prowlarr_indexer_unavailable gauge
		prowlarr_indexer_unavailable{indexer="Server.four",url="http://localhost:9117"} 1
		prowlarr_indexer_unavailable{indexer="Server1",url="http://localhost:9117"} 1
		prowlarr_indexer_unavailable{indexer="ServerTHREE",url="http://localhost:9117"} 1
		prowlarr_indexer_unavailable{indexer="ServerTwo",url="http://localhost:9117"} 1
		`)
	err := testutil.CollectAndCompare(testCol, expected)
	require.NoError(err)
}
