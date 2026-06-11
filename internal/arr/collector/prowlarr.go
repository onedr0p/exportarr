package collector

import (
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

// mergeIndexerStats folds a windowed indexer sample into the accumulated
// entry: counters add up, the response time is the latest observation.
func mergeIndexerStats(prev, next model.IndexerStats) model.IndexerStats {
	prev.Name = next.Name
	prev.AverageResponseTime = next.AverageResponseTime
	prev.NumberOfQueries += next.NumberOfQueries
	prev.NumberOfGrabs += next.NumberOfGrabs
	prev.NumberOfRssQueries += next.NumberOfRssQueries
	prev.NumberOfAuthQueries += next.NumberOfAuthQueries
	prev.NumberOfFailedQueries += next.NumberOfFailedQueries
	prev.NumberOfFailedGrabs += next.NumberOfFailedGrabs
	prev.NumberOfFailedRssQueries += next.NumberOfFailedRssQueries
	prev.NumberOfFailedAuthQueries += next.NumberOfFailedAuthQueries
	return prev
}

// mergeUserAgentStats folds a windowed user-agent sample into the accumulated
// entry.
func mergeUserAgentStats(prev, next model.UserAgentStats) model.UserAgentStats {
	prev.UserAgent = next.UserAgent
	prev.NumberOfQueries += next.NumberOfQueries
	prev.NumberOfGrabs += next.NumberOfGrabs
	return prev
}

// UnavailableIndexerEmitter emits prowlarr_indexer_unavailable from indexer
// status health messages.
type UnavailableIndexerEmitter struct {
	desc *prometheus.Desc
}

// NewUnavailableIndexerEmitter builds an emitter for the given prowlarr URL.
func NewUnavailableIndexerEmitter(url string) *UnavailableIndexerEmitter {
	return &UnavailableIndexerEmitter{
		desc: newDesc("prowlarr", "indexer_unavailable", "Indexers marked unavailable due to repeated errors", []string{"indexer"}, url),
	}
}

// Describe returns the emitter's metric descriptor.
func (e *UnavailableIndexerEmitter) Describe() *prometheus.Desc {
	return e.desc
}

// Emit derives unavailable-indexer metrics from a health message.
func (e *UnavailableIndexerEmitter) Emit(msg model.SystemHealthMessage) []prometheus.Metric {
	ret := []prometheus.Metric{}
	if msg.Source == "IndexerStatusCheck" || msg.Source == "IndexerLongTermStatusCheck" {
		parts := strings.Split(msg.Message, ":")
		svrs := parts[len(parts)-1]
		for svr := range strings.SplitSeq(svrs, ",") {
			ret = append(ret, prometheus.MustNewConstMetric(
				e.desc,
				prometheus.GaugeValue,
				1,
				strings.TrimSpace(svr),
			))
		}
	}
	return ret
}

type prowlarrCollector struct {
	client                           *client.Client
	config                           *config.ArrConfig                // App configuration
	indexerStatCache                 *statCache[model.IndexerStats]   // Cache of indexer stats
	userAgentStatCache               *statCache[model.UserAgentStats] // Cache of user agent stats
	statsMu                          sync.Mutex                       // Serializes the stats window so concurrent scrapes never double-count
	lastStatUpdate                   time.Time                        // Last time stat caches were updated
	indexerMetric                    *prometheus.Desc                 // Total number of configured indexers
	indexerEnabledMetric             *prometheus.Desc                 // Total number of enabled indexers
	indexerAverageResponseTimeMetric *prometheus.Desc                 // Average response time of indexers in ms
	indexerQueriesMetric             *prometheus.Desc                 // Total number of queries
	indexerGrabsMetric               *prometheus.Desc                 // Total number of grabs
	indexerRssQueriesMetric          *prometheus.Desc                 // Total number of rss queries
	indexerAuthQueriesMetric         *prometheus.Desc                 // Total number of auth queries
	indexerFailedQueriesMetric       *prometheus.Desc                 // Total number of failed queries
	indexerFailedGrabsMetric         *prometheus.Desc                 // Total number of failed grabs
	indexerFailedRssQueriesMetric    *prometheus.Desc                 // Total number of failed rss queries
	indexerFailedAuthQueriesMetric   *prometheus.Desc                 // Total number of failed auth queries
	indexerVipExpirationMetric       *prometheus.Desc                 // VIP expiration date
	userAgentMetric                  *prometheus.Desc                 // Total number of active user agents
	userAgentQueriesMetric           *prometheus.Desc                 // Total number of queries
	userAgentGrabsMetric             *prometheus.Desc                 // Total number of grabs
	errorMetric                      *prometheus.Desc                 // Error Description for use with InvalidMetric

}

// NewProwlarrCollector builds a collector for prowlarr indexer statistics.
func NewProwlarrCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	lastStatUpdate := time.Now()
	if c.Prowlarr.Backfill || !c.Prowlarr.BackfillSinceTime.IsZero() {
		lastStatUpdate = c.Prowlarr.BackfillSinceTime
	}
	return &prowlarrCollector{
		client:                           httpClient,
		config:                           c,
		indexerStatCache:                 newStatCache(mergeIndexerStats),
		userAgentStatCache:               newStatCache(mergeUserAgentStats),
		lastStatUpdate:                   lastStatUpdate,
		indexerMetric:                    newDesc("prowlarr", "indexer_total", "Total number of configured indexers", nil, c.URL),
		indexerEnabledMetric:             newDesc("prowlarr", "indexer_enabled_total", "Total number of enabled indexers", nil, c.URL),
		indexerAverageResponseTimeMetric: newDesc("prowlarr", "indexer_average_response_time_ms", "Average response time of indexers in ms", []string{"indexer"}, c.URL),
		indexerQueriesMetric:             newDesc("prowlarr", "indexer_queries_total", "Total number of queries", []string{"indexer"}, c.URL),
		indexerGrabsMetric:               newDesc("prowlarr", "indexer_grabs_total", "Total number of grabs", []string{"indexer"}, c.URL),
		indexerRssQueriesMetric:          newDesc("prowlarr", "indexer_rss_queries_total", "Total number of rss queries", []string{"indexer"}, c.URL),
		indexerAuthQueriesMetric:         newDesc("prowlarr", "indexer_auth_queries_total", "Total number of auth queries", []string{"indexer"}, c.URL),
		indexerFailedQueriesMetric:       newDesc("prowlarr", "indexer_failed_queries_total", "Total number of failed queries", []string{"indexer"}, c.URL),
		indexerFailedGrabsMetric:         newDesc("prowlarr", "indexer_failed_grabs_total", "Total number of failed grabs", []string{"indexer"}, c.URL),
		indexerFailedRssQueriesMetric:    newDesc("prowlarr", "indexer_failed_rss_queries_total", "Total number of failed rss queries", []string{"indexer"}, c.URL),
		indexerFailedAuthQueriesMetric:   newDesc("prowlarr", "indexer_failed_auth_queries_total", "Total number of failed auth queries", []string{"indexer"}, c.URL),
		indexerVipExpirationMetric:       newDesc("prowlarr", "indexer_vip_expires_in_seconds", "VIP expiration date", []string{"indexer"}, c.URL),
		userAgentMetric:                  newDesc("prowlarr", "user_agent_total", "Total number of active user agents", nil, c.URL),
		userAgentQueriesMetric:           newDesc("prowlarr", "user_agent_queries_total", "Total number of queries", []string{"user_agent"}, c.URL),
		userAgentGrabsMetric:             newDesc("prowlarr", "user_agent_grabs_total", "Total number of grabs", []string{"user_agent"}, c.URL),
		errorMetric:                      newDesc("prowlarr", "collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *prowlarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.indexerMetric
	ch <- collector.indexerAverageResponseTimeMetric
	ch <- collector.indexerQueriesMetric
	ch <- collector.indexerGrabsMetric
	ch <- collector.indexerRssQueriesMetric
	ch <- collector.indexerAuthQueriesMetric
	ch <- collector.indexerFailedQueriesMetric
	ch <- collector.indexerFailedGrabsMetric
	ch <- collector.indexerFailedRssQueriesMetric
	ch <- collector.indexerFailedAuthQueriesMetric
	ch <- collector.userAgentMetric
	ch <- collector.userAgentQueriesMetric
	ch <- collector.userAgentGrabsMetric
}

func (collector *prowlarrCollector) Collect(ch chan<- prometheus.Metric) {
	total := time.Now()
	log := slog.With("collector", "prowlarr")
	defer recoverCollect(log, ch, collector.errorMetric)
	c := collector.client

	enabledIndexers := 0

	indexers, err := client.Get[model.Indexer](c, "indexer")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting indexers", "error", err)
		return
	}
	for _, indexer := range indexers {
		if indexer.Enabled {
			enabledIndexers++
		}

		for _, field := range indexer.Fields {
			if field.Name != "vipExpiration" {
				continue
			}
			expiration, ok := field.Value.(string)
			if !ok || expiration == "" {
				continue
			}
			t, err := time.Parse("2006-01-02", expiration)
			if err != nil {
				// One malformed indexer should not abort the whole collection.
				log.Warn("Couldn't parse VIP Expiration", "indexer", indexer.Name, "error", err)
				continue
			}
			expirationSeconds := t.Unix() - time.Now().Unix()
			ch <- prometheus.MustNewConstMetric(collector.indexerVipExpirationMetric, prometheus.GaugeValue, float64(expirationSeconds), indexer.Name)
		}
	}

	// Hold the lock across read→fetch→accumulate: if two scrapes interleave
	// here, both fetch the same window and the deltas get counted twice.
	collector.statsMu.Lock()
	startDate := collector.lastStatUpdate.In(time.UTC)
	endDate := time.Now().In(time.UTC)

	params := client.QueryParams{}
	params.Add("startDate", startDate.Format(time.RFC3339))
	params.Add("endDate", endDate.Format(time.RFC3339))

	stats, err := client.Get[model.IndexerStatResponse](c, "indexerstats", params)
	if err != nil {
		collector.statsMu.Unlock()
		emitError(log, ch, collector.errorMetric, "Error getting indexer stats", "error", err)
		return
	}
	collector.lastStatUpdate = endDate

	for _, istats := range stats.Indexers {
		collector.indexerStatCache.Update(istats.Name, istats)
	}

	for _, cistats := range collector.indexerStatCache.Values() {
		ch <- prometheus.MustNewConstMetric(collector.indexerAverageResponseTimeMetric, prometheus.GaugeValue, float64(cistats.AverageResponseTime), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfQueries), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerGrabsMetric, prometheus.GaugeValue, float64(cistats.NumberOfGrabs), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerRssQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfRssQueries), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerAuthQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfAuthQueries), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerFailedQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfFailedQueries), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerFailedGrabsMetric, prometheus.GaugeValue, float64(cistats.NumberOfFailedGrabs), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerFailedRssQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfFailedRssQueries), cistats.Name)
		ch <- prometheus.MustNewConstMetric(collector.indexerFailedAuthQueriesMetric, prometheus.GaugeValue, float64(cistats.NumberOfFailedAuthQueries), cistats.Name)
	}

	for _, ustats := range stats.UserAgents {
		collector.userAgentStatCache.Update(ustats.UserAgent, ustats)
	}
	collector.statsMu.Unlock()

	for _, custats := range collector.userAgentStatCache.Values() {
		ch <- prometheus.MustNewConstMetric(collector.userAgentQueriesMetric, prometheus.GaugeValue, float64(custats.NumberOfQueries), custats.UserAgent)
		ch <- prometheus.MustNewConstMetric(collector.userAgentGrabsMetric, prometheus.GaugeValue, float64(custats.NumberOfGrabs), custats.UserAgent)
	}

	ch <- prometheus.MustNewConstMetric(collector.indexerMetric, prometheus.GaugeValue, float64(len(indexers)))
	ch <- prometheus.MustNewConstMetric(collector.userAgentMetric, prometheus.GaugeValue, float64(len(stats.UserAgents)))
	ch <- prometheus.MustNewConstMetric(collector.indexerEnabledMetric, prometheus.GaugeValue, float64(enabledIndexers))

	log.Debug("Prowlarr cycle completed", "duration", time.Since(total))
}
