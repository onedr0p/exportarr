package collector

import (
	"errors"
	"sync"

	"github.com/shamelin/exportarr/internal/sabnzbd/model"
)

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
		// If the day parsed is empty, it means there are no server side stats.
		// If we have exportarr stats, something likely went wrong,
		return s, errors.New("no Parsed Dates from Server, but cache is not empty")
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

type ServersStatsCache struct {
	lock    sync.RWMutex
	Total   int
	Servers map[string]serverStatCache
}

func NewServersStatsCache() *ServersStatsCache {
	return &ServersStatsCache{
		Servers: make(map[string]serverStatCache),
	}
}

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

func (c *ServersStatsCache) GetTotal() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.Total
}

func (c *ServersStatsCache) GetServerMap() map[string]ServerStats {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ret := make(map[string]ServerStats)
	for k, v := range c.Servers {
		ret[k] = v
	}

	return ret
}
