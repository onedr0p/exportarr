package collector

import (
	"github.com/onedr0p/exportarr/internal/assert"
	"testing"

	"github.com/onedr0p/exportarr/internal/sabnzbd/model"
)

func TestUpdateServerStatsCache_SameDay(t *testing.T) {
	cache := NewServersStatsCache()
	_ = cache.Update(model.ServerStats{
		Total: 1,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           1,
				ArticlesTried:   2,
				ArticlesSuccess: 2,
				DayParsed:       "2020-01-01",
			},
			"server2": {
				Total:           2,
				ArticlesTried:   4,
				ArticlesSuccess: 4,
				DayParsed:       "2020-01-01",
			},
		},
	})
	assert.Equal(t, cache.GetTotal(), 1)
	m := cache.GetServerMap()
	assert.Equal(t, len(m), 2)

	server1 := m["server1"]
	assert.Equal(t, server1.GetTotal(), 1)
	assert.Equal(t, server1.GetArticlesTried(), 2)
	assert.Equal(t, server1.GetArticlesSuccess(), 2)

	server2 := m["server2"]
	assert.Equal(t, server2.GetTotal(), 2)
	assert.Equal(t, server2.GetArticlesTried(), 4)
	assert.Equal(t, server2.GetArticlesSuccess(), 4)
	_ = cache.Update(model.ServerStats{
		Total: 2,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           2,
				ArticlesTried:   6,
				ArticlesSuccess: 6,
				DayParsed:       "2020-01-01",
			},
			"server2": {
				Total:           3,
				ArticlesTried:   8,
				ArticlesSuccess: 8,
				DayParsed:       "2020-01-01",
			},
		},
	})
	assert.Equal(t, cache.GetTotal(), 2)
	m = cache.GetServerMap()
	assert.Equal(t, len(m), 2)

	server1 = m["server1"]
	assert.Equal(t, server1.GetTotal(), 2)
	assert.Equal(t, server1.GetArticlesTried(), 6)
	assert.Equal(t, server1.GetArticlesSuccess(), 6)

	server2 = m["server2"]
	assert.Equal(t, server2.GetTotal(), 3)
	assert.Equal(t, server2.GetArticlesTried(), 8)
	assert.Equal(t, server2.GetArticlesSuccess(), 8)
}

func TestUpdateServerStatsCache_DifferentDay(t *testing.T) {
	cache := NewServersStatsCache()
	_ = cache.Update(model.ServerStats{
		Total: 1,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           1,
				ArticlesTried:   2,
				ArticlesSuccess: 2,
				DayParsed:       "2020-01-01",
			},
			"server2": {
				Total:           2,
				ArticlesTried:   4,
				ArticlesSuccess: 4,
				DayParsed:       "2020-01-01",
			},
		},
	})
	assert.Equal(t, cache.GetTotal(), 1)
	m := cache.GetServerMap()
	assert.Equal(t, len(m), 2)

	server1 := m["server1"]
	assert.Equal(t, server1.GetTotal(), 1)
	assert.Equal(t, server1.GetArticlesTried(), 2)
	assert.Equal(t, server1.GetArticlesSuccess(), 2)

	server2 := m["server2"]
	assert.Equal(t, server2.GetTotal(), 2)
	assert.Equal(t, server2.GetArticlesTried(), 4)
	assert.Equal(t, server2.GetArticlesSuccess(), 4)
	_ = cache.Update(model.ServerStats{
		Total: 2,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           2,
				ArticlesTried:   6,
				ArticlesSuccess: 6,
				DayParsed:       "2020-01-02",
			},
			"server2": {
				Total:           3,
				ArticlesTried:   8,
				ArticlesSuccess: 8,
				DayParsed:       "2020-01-02",
			},
		},
	})
	assert.Equal(t, cache.GetTotal(), 2)
	m = cache.GetServerMap()
	assert.Equal(t, len(m), 2)

	server1 = m["server1"]
	assert.Equal(t, server1.GetTotal(), 2)
	assert.Equal(t, server1.GetArticlesTried(), 8)
	assert.Equal(t, server1.GetArticlesSuccess(), 8)

	server2 = m["server2"]
	assert.Equal(t, server2.GetTotal(), 3)
	assert.Equal(t, server2.GetArticlesTried(), 12)
	assert.Equal(t, server2.GetArticlesSuccess(), 12)
}

func TestUpdateServerStatsCache_EmptyServerStats(t *testing.T) {
	tests := []struct {
		name          string
		startingStats model.ServerStats
		endingStats   model.ServerStats
		shouldError   bool
	}{
		{
			name: "Empty Starting Date",
			startingStats: model.ServerStats{
				Total:   1,
				Servers: map[string]model.ServerStat{},
			},
		},
		{
			name: "Non-Empty Starting Date",
			startingStats: model.ServerStats{
				Total: 1,
				Servers: map[string]model.ServerStat{
					"server1": {
						Total:           1,
						ArticlesTried:   2,
						ArticlesSuccess: 2,
						DayParsed:       "2020-01-01",
					},
				},
			},
			// A server-side stats reset self-heals instead of erroring.
			shouldError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewServersStatsCache()
			err := cache.Update(tt.startingStats)
			assert.NoError(t, err)
			err = cache.Update(model.ServerStats{
				Total: 1,
				Servers: map[string]model.ServerStat{
					"server1": {
						Total:     1,
						DayParsed: "",
					},
				},
			})
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewServerStatsCache_SetsServers(t *testing.T) {
	cache := NewServersStatsCache()
	assert.NotNil(t, cache.Servers)
}

func TestUpdateServerStatsCache(t *testing.T) {
	cache := NewServersStatsCache()
	_ = cache.Update(model.ServerStats{
		Total: 1,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           1,
				ArticlesTried:   2,
				ArticlesSuccess: 2,
				DayParsed:       "2020-01-01",
			},
			"server2": {
				Total:           2,
				ArticlesTried:   4,
				ArticlesSuccess: 4,
				DayParsed:       "2020-01-01",
			},
		},
	})

	server1 := cache.Servers["server1"]
	server2 := cache.Servers["server2"]

	assert.Equal(t, server1.GetTotal(), 1)
	assert.Equal(t, server1.GetArticlesTried(), 2)
	assert.Equal(t, server1.GetArticlesSuccess(), 2)
	assert.Equal(t, server2.GetTotal(), 2)
	assert.Equal(t, server2.GetArticlesTried(), 4)
	assert.Equal(t, server2.GetArticlesSuccess(), 4)

	_ = cache.Update(model.ServerStats{
		Total: 2,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           3,
				ArticlesTried:   6,
				ArticlesSuccess: 6,
				DayParsed:       "2020-01-01",
			},
		},
	})

	server1 = cache.Servers["server1"]
	server2 = cache.Servers["server2"]

	assert.Equal(t, cache.GetTotal(), 2)
	assert.Equal(t, server1.GetTotal(), 3)
	assert.Equal(t, server1.GetArticlesTried(), 6)
	assert.Equal(t, server1.GetArticlesSuccess(), 6)
	assert.Equal(t, server2.GetTotal(), 2)
	assert.Equal(t, server2.GetArticlesTried(), 4)
	assert.Equal(t, server2.GetArticlesSuccess(), 4)
}

func TestGetServerMap_ReturnsCopy(t *testing.T) {
	// It's important to return a true copy to maintain thread safety

	cache := NewServersStatsCache()
	_ = cache.Update(model.ServerStats{
		Total: 1,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           1,
				ArticlesTried:   2,
				ArticlesSuccess: 2,
				DayParsed:       "2020-01-01",
			},
		},
	})

	serverMap := cache.GetServerMap()

	for k, v := range serverMap {
		assert.DeepEqual(t, v, cache.Servers[k])
	}

	// GetServerMap must return a copy: writes to it must not leak into the
	// cache's internal map.
	serverMap["sentinel"] = serverStatCache{}
	_, leaked := cache.Servers["sentinel"]
	assert.False(t, leaked, "GetServerMap must return a copy")
	delete(serverMap, "sentinel")

	_ = cache.Update(model.ServerStats{
		Total: 2,
		Servers: map[string]model.ServerStat{
			"server1": {
				Total:           3,
				ArticlesTried:   6,
				ArticlesSuccess: 6,
				DayParsed:       "2020-01-01",
			},
		},
	})

	cServer := cache.Servers["server1"]
	sServer := serverMap["server1"]

	assert.NotEqual(t, sServer.GetTotal(), cServer.GetTotal())
	assert.NotEqual(t, sServer.GetArticlesTried(), cServer.GetArticlesTried())
	assert.NotEqual(t, sServer.GetArticlesSuccess(), cServer.GetArticlesSuccess())
}
