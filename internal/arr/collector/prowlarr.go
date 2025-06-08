package collector

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type indexerStatCache struct {
	cache map[string]model.IndexerStats
	mutex sync.Mutex
}

func NewIndexerStatCache() indexerStatCache {
	return indexerStatCache{
		cache: make(map[string]model.IndexerStats),
	}
}

func (i *indexerStatCache) UpdateKey(key string, value model.IndexerStats) model.IndexerStats {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	entry, ok := i.cache[key]
	if !ok {
		entry = model.IndexerStats{
			Name: value.Name,
		}
	}
	entry.AverageResponseTime = value.AverageResponseTime
	entry.NumberOfQueries += value.NumberOfQueries
	entry.NumberOfGrabs += value.NumberOfGrabs
	entry.NumberOfRssQueries += value.NumberOfRssQueries
	entry.NumberOfAuthQueries += value.NumberOfAuthQueries
	entry.NumberOfFailedQueries += value.NumberOfFailedQueries
	entry.NumberOfFailedGrabs += value.NumberOfFailedGrabs
	entry.NumberOfFailedRssQueries += value.NumberOfFailedRssQueries
	entry.NumberOfFailedAuthQueries += value.NumberOfFailedAuthQueries
	i.cache[key] = entry
	return entry
}

func (i *indexerStatCache) GetIndexerStats() []model.IndexerStats {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	ret := make([]model.IndexerStats, 0, len(i.cache))
	for _, v := range i.cache {
		ret = append(ret, v)
	}
	return ret
}

type userAgentStatCache struct {
	cache map[string]model.UserAgentStats
	mutex sync.Mutex
}

func NewUserAgentCache() userAgentStatCache {
	return userAgentStatCache{
		cache: make(map[string]model.UserAgentStats),
	}
}

func (u *userAgentStatCache) GetUserAgentStats() []model.UserAgentStats {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	ret := make([]model.UserAgentStats, 0, len(u.cache))
	for _, v := range u.cache {
		ret = append(ret, v)
	}
	return ret
}

func (u *userAgentStatCache) UpdateKey(key string, value model.UserAgentStats) model.UserAgentStats {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	entry, ok := u.cache[key]
	if !ok {
		entry = model.UserAgentStats{
			UserAgent: value.UserAgent,
		}
	}

	entry.NumberOfQueries += value.NumberOfQueries
	entry.NumberOfGrabs += value.NumberOfGrabs
	u.cache[key] = entry
	return entry
}

type UnavailableIndexerEmitter struct {
	url string
}

func NewUnavailableIndexerEmitter(url string) *UnavailableIndexerEmitter {
	return &UnavailableIndexerEmitter{
		url: url,
	}
}

func (e *UnavailableIndexerEmitter) Describe() *prometheus.Desc {
	return prometheus.NewDesc(
		"prowlarr_indexer_unavailable",
		"Indexers marked unavailable due to repeated errors",
		[]string{"indexer"},
		prometheus.Labels{"url": e.url},
	)
}

func (e *UnavailableIndexerEmitter) Emit(msg model.SystemHealthMessage) []prometheus.Metric {
	ret := []prometheus.Metric{}
	if msg.Source == "IndexerStatusCheck" || msg.Source == "IndexerLongTermStatusCheck" {
		parts := strings.Split(msg.Message, ":")
		svrs := parts[len(parts)-1]
		for _, svr := range strings.Split(svrs, ",") {
			ret = append(ret, prometheus.MustNewConstMetric(
				e.Describe(),
				prometheus.GaugeValue,
				1,
				strings.TrimSpace(svr),
			))
		}
	}
	return ret
}

type prowlarrCollector struct {
	config                           *config.ArrConfig  // App configuration
	indexerStatCache                 indexerStatCache   // Cache of indexer stats
	userAgentStatCache               userAgentStatCache // Cache of user agent stats
	lastStatUpdate                   time.Time          // Last time stat caches were updated
	indexerMetric                    *prometheus.Desc   // Total number of configured indexers
	indexerEnabledMetric             *prometheus.Desc   // Total number of enabled indexers
	indexerAverageResponseTimeMetric *prometheus.Desc   // Average response time of indexers in ms
	indexerQueriesMetric             *prometheus.Desc   // Total number of queries
	indexerGrabsMetric               *prometheus.Desc   // Total number of grabs
	indexerRssQueriesMetric          *prometheus.Desc   // Total number of rss queries
	indexerAuthQueriesMetric         *prometheus.Desc   // Total number of auth queries
	indexerFailedQueriesMetric       *prometheus.Desc   // Total number of failed queries
	indexerFailedGrabsMetric         *prometheus.Desc   // Total number of failed grabs
	indexerFailedRssQueriesMetric    *prometheus.Desc   // Total number of failed rss queries
	indexerFailedAuthQueriesMetric   *prometheus.Desc   // Total number of failed auth queries
	indexerVipExpirationMetric       *prometheus.Desc   // VIP expiration date
	userAgentMetric                  *prometheus.Desc   // Total number of active user agents
	userAgentQueriesMetric           *prometheus.Desc   // Total number of queries
	userAgentGrabsMetric             *prometheus.Desc   // Total number of grabs
	errorMetric                      *prometheus.Desc   // Error Description for use with InvalidMetric

}

func NewProwlarrCollector(c *config.ArrConfig) *prowlarrCollector {
	lastStatUpdate := time.Now()
	if c.Prowlarr.Backfill || !c.Prowlarr.BackfillSinceTime.IsZero() {
		lastStatUpdate = c.Prowlarr.BackfillSinceTime
	}
	return &prowlarrCollector{
		config:             c,
		indexerStatCache:   NewIndexerStatCache(),
		userAgentStatCache: NewUserAgentCache(),
		lastStatUpdate:     lastStatUpdate,
		indexerMetric: prometheus.NewDesc(
			"prowlarr_indexer_total",
			"Total number of configured indexers",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		indexerEnabledMetric: prometheus.NewDesc(
			"prowlarr_indexer_enabled_total",
			"Total number of enabled indexers",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		indexerAverageResponseTimeMetric: prometheus.NewDesc(
			"prowlarr_indexer_average_response_time_ms",
			"Average response time of indexers in ms",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_queries_total",
			"Total number of queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerGrabsMetric: prometheus.NewDesc(
			"prowlarr_indexer_grabs_total",
			"Total number of grabs",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerRssQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_rss_queries_total",
			"Total number of rss queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerAuthQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_auth_queries_total",
			"Total number of auth queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerFailedQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_queries_total",
			"Total number of failed queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerFailedGrabsMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_grabs_total",
			"Total number of failed grabs",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerFailedRssQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_rss_queries_total",
			"Total number of failed rss queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerFailedAuthQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_auth_queries_total",
			"Total number of failed auth queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		indexerVipExpirationMetric: prometheus.NewDesc(
			"prowlarr_indexer_vip_expires_in_seconds",
			"VIP expiration date",
			[]string{"indexer"},
			prometheus.Labels{"url": c.URL},
		),
		userAgentMetric: prometheus.NewDesc(
			"prowlarr_user_agent_total",
			"Total number of active user agents",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		userAgentQueriesMetric: prometheus.NewDesc(
			"prowlarr_user_agent_queries_total",
			"Total number of queries",
			[]string{"user_agent"},
			prometheus.Labels{"url": c.URL},
		),
		userAgentGrabsMetric: prometheus.NewDesc(
			"prowlarr_user_agent_grabs_total",
			"Total number of grabs",
			[]string{"user_agent"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			"prowlarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *prowlarrCollector) Describe(ch chan<- *prometheus.Desc) {
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
	log := zap.S().With("collector", "prowlarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorf("Error creating client: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	var enabledIndexers = 0

	indexers := model.Indexer{}
	if err := c.DoRequest("indexer", &indexers); err != nil {
		log.Errorf("Error getting indexers: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, indexer := range indexers {
		if indexer.Enabled {
			enabledIndexers++
		}

		for _, field := range indexer.Fields {
			if field.Name == "vipExpiration" && field.Value != "" {
				t, err := time.Parse("2006-01-02", field.Value.(string))
				if err != nil {
					log.Errorf("Couldn't parse VIP Expiration: %s", err)
					ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
					return
				}
				expirationSeconds := t.Unix() - time.Now().Unix()
				ch <- prometheus.MustNewConstMetric(collector.indexerVipExpirationMetric, prometheus.GaugeValue, float64(expirationSeconds), indexer.Name)
			}
		}
	}

	stats := model.IndexerStatResponse{}
	startDate := collector.lastStatUpdate.In(time.UTC)
	endDate := time.Now().In(time.UTC)

	params := client.QueryParams{}
	params.Add("startDate", startDate.Format(time.RFC3339))
	params.Add("endDate", endDate.Format(time.RFC3339))

	if err := c.DoRequest("indexerstats", &stats, params); err != nil {
		log.Errorf("Error getting indexer stats: %s", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	collector.lastStatUpdate = endDate

	for _, istats := range stats.Indexers {
		collector.indexerStatCache.UpdateKey(istats.Name, istats)
	}

	for _, cistats := range collector.indexerStatCache.GetIndexerStats() {
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
		collector.userAgentStatCache.UpdateKey(ustats.UserAgent, ustats)
	}

	for _, custats := range collector.userAgentStatCache.GetUserAgentStats() {
		ch <- prometheus.MustNewConstMetric(collector.userAgentQueriesMetric, prometheus.GaugeValue, float64(custats.NumberOfQueries), custats.UserAgent)
		ch <- prometheus.MustNewConstMetric(collector.userAgentGrabsMetric, prometheus.GaugeValue, float64(custats.NumberOfGrabs), custats.UserAgent)
	}

	ch <- prometheus.MustNewConstMetric(collector.indexerMetric, prometheus.GaugeValue, float64(len(indexers)))
	ch <- prometheus.MustNewConstMetric(collector.userAgentMetric, prometheus.GaugeValue, float64(len(stats.UserAgents)))
	ch <- prometheus.MustNewConstMetric(collector.indexerEnabledMetric, prometheus.GaugeValue, float64(enabledIndexers))

	log.Debugf("TIME :: total took %s ",
		time.Since(total).String(),
	)
}
