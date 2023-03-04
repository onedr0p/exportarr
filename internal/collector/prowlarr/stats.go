package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
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

type prowlarrCollector struct {
	config                           *cli.Context       // App configuration
	configFile                       *model.Config      // *arr configuration from config.xml
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
}

func NewProwlarrCollector(c *cli.Context, cf *model.Config) *prowlarrCollector {
	var lastStatUpdate time.Time
	if c.Bool("enable-additional-metrics") {
		// If additional metrics are enabled, backfill the cache.
		lastStatUpdate = time.Time{}
	} else {
		lastStatUpdate = time.Now()
	}
	return &prowlarrCollector{
		config:             c,
		configFile:         cf,
		indexerStatCache:   NewIndexerStatCache(),
		userAgentStatCache: NewUserAgentCache(),
		lastStatUpdate:     lastStatUpdate,
		indexerMetric: prometheus.NewDesc(
			"prowlarr_indexer_total",
			"Total number of configured indexers",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		indexerEnabledMetric: prometheus.NewDesc(
			"prowlarr_indexer_enabled_total",
			"Total number of enabled indexers",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		indexerAverageResponseTimeMetric: prometheus.NewDesc(
			"prowlarr_indexer_average_response_time_ms",
			"Average response time of indexers in ms",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_queries_total",
			"Total number of queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerGrabsMetric: prometheus.NewDesc(
			"prowlarr_indexer_grabs_total",
			"Total number of grabs",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerRssQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_rss_queries_total",
			"Total number of rss queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerAuthQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_auth_queries_total",
			"Total number of auth queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerFailedQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_queries_total",
			"Total number of failed queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerFailedGrabsMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_grabs_total",
			"Total number of failed grabs",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerFailedRssQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_rss_queries_total",
			"Total number of failed rss queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerFailedAuthQueriesMetric: prometheus.NewDesc(
			"prowlarr_indexer_failed_auth_queries_total",
			"Total number of failed auth queries",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		indexerVipExpirationMetric: prometheus.NewDesc(
			"prowlarr_indexer_vip_expires_in_seconds",
			"VIP expiration date",
			[]string{"indexer"},
			prometheus.Labels{"url": c.String("url")},
		),
		userAgentMetric: prometheus.NewDesc(
			"prowlarr_user_agent_total",
			"Total number of active user agents",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		userAgentQueriesMetric: prometheus.NewDesc(
			"prowlarr_user_agent_queries_total",
			"Total number of queries",
			[]string{"user_agent"},
			prometheus.Labels{"url": c.String("url")},
		),
		userAgentGrabsMetric: prometheus.NewDesc(
			"prowlarr_user_agent_grabs_total",
			"Total number of grabs",
			[]string{"user_agent"},
			prometheus.Labels{"url": c.String("url")},
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
	c := client.NewClient(collector.config, collector.configFile)

	var enabledIndexers = 0

	indexers := model.Indexer{}
	if err := c.DoRequest("indexer", &indexers); err != nil {
		log.Fatal(err)
	}
	for _, indexer := range indexers {
		if indexer.Enabled {
			enabledIndexers++
		}

		for _, field := range indexer.Fields {
			if field.Name == "vipExpiration" && field.Value != "" {
				t, err := time.Parse("2006-01-02", field.Value.(string))
				if err != nil {
					log.Fatal(err)
				}
				expirationSeconds := t.Unix() - time.Now().Unix()
				ch <- prometheus.MustNewConstMetric(collector.indexerVipExpirationMetric, prometheus.GaugeValue, float64(expirationSeconds), indexer.Name)
			}
		}
	}

	stats := model.IndexerStatResponse{}
	startDate := collector.lastStatUpdate.In(time.UTC)
	endDate := time.Now().In(time.UTC)
	req := fmt.Sprintf("indexerstats?startDate=%s&endDate=%s", startDate.Format(time.RFC3339), endDate.Format(time.RFC3339))
	if err := c.DoRequest(req, &stats); err != nil {
		log.Fatal(err)
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

	log.Debug("TIME :: total took %s ",
		time.Since(total),
	)
}
