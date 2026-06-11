// Package collector implements the SABnzbd Prometheus collector.
package collector

import (
	"sync"

	"github.com/onedr0p/exportarr/internal/sabnzbd/model"
)

// ServerStats is a read-only view over accumulated per-server statistics.
type ServerStats interface {
	Update(stat model.ServerStat) (ServerStats, error)
	GetTotal() int
	GetArticlesTried() int
	GetArticlesSuccess() int
}

type serverStatCache struct {
	total                     int
	articlesTriedHistorical   int
	articlesTriedToday        int
	articlesSuccessHistorical int
	articlesSuccessToday      int
	todayKey                  string
}

func (s serverStatCache) Update(stat model.ServerStat) (ServerStats, error) {
	if stat.DayParsed == "" && s.todayKey != "" {
		// SABnzbd's per-day stats disappeared (history reset server-side). Start
		// accumulating again rather than failing every scrape until restart.
		return serverStatCache{total: stat.Total}, nil
	}
	s.total = stat.Total

	if stat.DayParsed != s.todayKey {
		s.articlesTriedHistorical += s.articlesTriedToday
		s.articlesSuccessHistorical += s.articlesSuccessToday
		s.articlesTriedToday = 0
		s.articlesSuccessToday = 0
		s.todayKey = stat.DayParsed
	}

	s.articlesTriedToday = stat.ArticlesTried
	s.articlesSuccessToday = stat.ArticlesSuccess

	return s, nil
}

func (s serverStatCache) GetTotal() int {
	return s.total
}

func (s serverStatCache) GetArticlesTried() int {
	return s.articlesTriedHistorical + s.articlesTriedToday
}

func (s serverStatCache) GetArticlesSuccess() int {
	return s.articlesSuccessHistorical + s.articlesSuccessToday
}

// ServersStatsCache accumulates per-server statistics across scrapes so
// counters survive SABnzbd's daily rollover.
type ServersStatsCache struct {
	lock    sync.RWMutex
	Total   int
	Servers map[string]serverStatCache
}

// NewServersStatsCache returns an empty ServersStatsCache.
func NewServersStatsCache() *ServersStatsCache {
	return &ServersStatsCache{
		Servers: make(map[string]serverStatCache),
	}
}

// Update folds a fresh ServerStats sample into the cache.
func (c *ServersStatsCache) Update(stats model.ServerStats) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.Total = stats.Total

	for name, srv := range stats.Servers {
		var toCache serverStatCache
		if cached, ok := c.Servers[name]; ok {
			toCache = cached
		}

		updated, err := toCache.Update(srv)
		if err != nil {
			return err
		}
		c.Servers[name] = updated.(serverStatCache)
	}
	return nil
}

// GetTotal returns the total bytes downloaded across all servers.
func (c *ServersStatsCache) GetTotal() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.Total
}

// GetServerMap returns a copy of the per-server statistics.
func (c *ServersStatsCache) GetServerMap() map[string]ServerStats {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ret := make(map[string]ServerStats)
	for k, v := range c.Servers {
		ret[k] = v
	}

	return ret
}
