package collector

import (
	"testing"

	"github.com/shamelin/exportarr/internal/sabnzbd/model"
	"github.com/stretchr/testify/require"
)

func TestUpdateServerStatsCache_SameDay(t *testing.T) {
	require := require.New(t)
	cache := NewServersStatsCache()
	cache.Update(model.ServerStats{
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
	require.Equal(1, cache.GetTotal())
	m := cache.GetServerMap()
	require.Equal(2, len(m))

	server1 := m["server1"]
	require.Equal(1, server1.GetTotal())
	require.Equal(2, server1.GetArticlesTried())
	require.Equal(2, server1.GetArticlesSuccess())

	server2 := m["server2"]
	require.Equal(2, server2.GetTotal())
	require.Equal(4, server2.GetArticlesTried())
	require.Equal(4, server2.GetArticlesSuccess())
	cache.Update(model.ServerStats{
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
	require.Equal(2, cache.GetTotal())
	m = cache.GetServerMap()
	require.Equal(2, len(m))

	server1 = m["server1"]
	require.Equal(2, server1.GetTotal())
	require.Equal(6, server1.GetArticlesTried())
	require.Equal(6, server1.GetArticlesSuccess())

	server2 = m["server2"]
	require.Equal(3, server2.GetTotal())
	require.Equal(8, server2.GetArticlesTried())
	require.Equal(8, server2.GetArticlesSuccess())
}

func TestUpdateServerStatsCache_DifferentDay(t *testing.T) {
	require := require.New(t)
	cache := NewServersStatsCache()
	cache.Update(model.ServerStats{
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
	require.Equal(1, cache.GetTotal())
	m := cache.GetServerMap()
	require.Equal(2, len(m))

	server1 := m["server1"]
	require.Equal(1, server1.GetTotal())
	require.Equal(2, server1.GetArticlesTried())
	require.Equal(2, server1.GetArticlesSuccess())

	server2 := m["server2"]
	require.Equal(2, server2.GetTotal())
	require.Equal(4, server2.GetArticlesTried())
	require.Equal(4, server2.GetArticlesSuccess())
	cache.Update(model.ServerStats{
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
	require.Equal(2, cache.GetTotal())
	m = cache.GetServerMap()
	require.Equal(2, len(m))

	server1 = m["server1"]
	require.Equal(2, server1.GetTotal())
	require.Equal(8, server1.GetArticlesTried())
	require.Equal(8, server1.GetArticlesSuccess())

	server2 = m["server2"]
	require.Equal(3, server2.GetTotal())
	require.Equal(12, server2.GetArticlesTried())
	require.Equal(12, server2.GetArticlesSuccess())
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
			shouldError: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			cache := NewServersStatsCache()
			err := cache.Update(tt.startingStats)
			require.NoError(err)
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
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

func TestNewServerStatsCache_SetsServers(t *testing.T) {
	require := require.New(t)
	cache := NewServersStatsCache()
	require.NotNil(cache.Servers)
}

func TestUpdateServerStatsCache(t *testing.T) {
	require := require.New(t)
	cache := NewServersStatsCache()
	cache.Update(model.ServerStats{
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

	require.Equal(1, server1.GetTotal())
	require.Equal(2, server1.GetArticlesTried())
	require.Equal(2, server1.GetArticlesSuccess())
	require.Equal(2, server2.GetTotal())
	require.Equal(4, server2.GetArticlesTried())
	require.Equal(4, server2.GetArticlesSuccess())

	cache.Update(model.ServerStats{
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

	require.Equal(2, cache.GetTotal())
	require.Equal(3, server1.GetTotal())
	require.Equal(6, server1.GetArticlesTried())
	require.Equal(6, server1.GetArticlesSuccess())
	require.Equal(2, server2.GetTotal())
	require.Equal(4, server2.GetArticlesTried())
	require.Equal(4, server2.GetArticlesSuccess())
}

func TestGetServerMap_ReturnsCopy(t *testing.T) {
	// It's important to return a true copy to maintain thread safety
	require := require.New(t)

	cache := NewServersStatsCache()
	cache.Update(model.ServerStats{
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
		require.Equal(cache.Servers[k], v)
	}

	require.NotSame(&cache.Servers, &serverMap)

	cache.Update(model.ServerStats{
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

	require.NotEqual(cServer.GetTotal(), sServer.GetTotal())
	require.NotEqual(cServer.GetArticlesTried(), sServer.GetArticlesTried())
	require.NotEqual(cServer.GetArticlesSuccess(), sServer.GetArticlesSuccess())
}
