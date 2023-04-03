package model

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type TestServerStatsResponse struct {
	Total   int                               `json:"total"` // Total Data Downloaded in bytes
	Servers map[string]TestServerStatResponse `json:"servers"`
}

type TestServerStatResponse struct {
	Total           int            `json:"total"` // Total Data Downloaded in bytes
	ArticlesTried   map[string]int `json:"articles_tried"`
	ArticlesSuccess map[string]int `json:"articles_success"`
}

func TestStatusToString(t *testing.T) {
	require := require.New(t)
	require.Equal("Downloading", DOWNLOADING.String())
	require.Equal("Paused", PAUSED.String())
	require.Equal("Idle", IDLE.String())
	require.Equal("Unknown", Status(999).String())
}

func TestStatusFromString(t *testing.T) {
	require := require.New(t)
	require.Equal(DOWNLOADING, StatusFromString("Downloading"))
	require.Equal(PAUSED, StatusFromString("Paused"))
	require.Equal(IDLE, StatusFromString("Idle"))
	require.Equal(UNKNOWN, StatusFromString("Unknown"))
	require.Equal(UNKNOWN, StatusFromString("Unknown"))
}

func TestStatusToFloat(t *testing.T) {
	require := require.New(t)
	require.Equal(3.0, DOWNLOADING.Float64())
	require.Equal(2.0, PAUSED.Float64())
	require.Equal(1.0, IDLE.Float64())
	require.Equal(0.0, UNKNOWN.Float64())
}

func TestQueueStats_UnmarshalJSON(t *testing.T) {
	require := require.New(t)
	queue, err := os.ReadFile("../test_fixtures/queue.json")
	require.NoError(err)

	var queueStats QueueStats
	err = queueStats.UnmarshalJSON(queue)
	require.NoError(err)
	require.Equal("3.7.2", queueStats.Version)
	require.False(queueStats.Paused)
	require.False(queueStats.PausedAll)
	require.Equal(time.Duration(0), queueStats.PauseDuration)
	require.Equal(3.64627623936e+10, queueStats.DownloadDirDiskspaceUsed)
	require.Equal(4.4971327488e+10, queueStats.DownloadDirDiskspaceTotal)
	require.Equal(3.64061392896e+10, queueStats.CompletedDirDiskspaceUsed)
	require.Equal(4.4972376064e+10, queueStats.CompletedDirDiskspaceTotal)
	require.Equal(100.0, queueStats.SpeedLimit)
	require.Equal(1.048576e+09, queueStats.SpeedLimitAbs)
	require.Equal(0.0, queueStats.HaveWarnings)
	require.Equal(1.07911053312e+12, queueStats.Quota)
	require.True(queueStats.HaveQuota)
	require.Equal(1.073741824e+12, queueStats.RemainingQuota)
	require.Equal(0.0, queueStats.CacheArt)
	require.Equal(0.0, queueStats.CacheSize)
	require.Equal(358.4, queueStats.Speed)
	require.Equal(3.21070825472e+09, queueStats.RemainingSize)
	require.Equal(3.21175683072e+09, queueStats.Size)
	require.Equal(2.0, queueStats.ItemsInQueue)
	require.Equal(DOWNLOADING, queueStats.Status)
	d, _ := time.ParseDuration("2495h59m3s")
	require.Equal(d, queueStats.TimeEstimate)

}

func TestQueueStats_ParseSize(t *testing.T) {
	parameters := []struct {
		input    string
		expected float64
	}{
		{"0 B", 0.0},
		{"1 B", 1.0},
		{"1.0 B", 1.0},
		{"10 K", 10240.0},
		{"10.0 KB", 10240.0},
		{"10 M", 10485760.0},
		{"10.0 MB", 10485760.0},
		{"10 G", 10737418240.0},
		{"10.0 GB", 10737418240.0},
		{"10 T", 10995116277760.0},
		{"10.0 TB", 10995116277760.0},
		{"10 P", 11258999068426240.0},
		{"10.0 PB", 11258999068426240.0},
	}

	require := require.New(t)

	for _, parameter := range parameters {
		statsResponse := fmt.Sprintf(`{"queue": {"left_quota": "%s"}}`, parameter.input)
		var stats QueueStats
		err := json.Unmarshal([]byte(statsResponse), &stats)
		require.NoError(err)
		require.Equal(parameter.expected, stats.RemainingQuota)
	}
}

func TestQueueStatus_ParseDuration(t *testing.T) {
	parameters := []struct {
		input    string
		expected time.Duration
	}{
		{"", time.Duration(0)},
		{"10", time.Duration(10) * time.Second},
		{"10:01", time.Duration(10)*time.Minute + time.Duration(1)*time.Second},
		{"13:12:11", time.Duration(13)*time.Hour + time.Duration(12)*time.Minute + time.Duration(11)*time.Second},
		{"14:13:12:11", time.Duration(349)*time.Hour + time.Duration(12)*time.Minute + time.Duration(11)*time.Second},
	}

	require := require.New(t)

	for _, parameter := range parameters {
		var stats QueueStats
		statsResponse := fmt.Sprintf(`{ "queue": {"timeleft": "%s"}}`, parameter.input)
		err := json.Unmarshal([]byte(statsResponse), &stats)
		require.NoError(err)
		require.Equal(parameter.expected, stats.TimeEstimate)
	}
}

func TestServerStats_UnmarshalJSON(t *testing.T) {
	require := require.New(t)

	statsResponse := TestServerStatsResponse{
		Total: 123456789,
		Servers: map[string]TestServerStatResponse{
			"server1": {
				Total: 234567890,
				ArticlesTried: map[string]int{
					"2020-01-01": 1,
					"2020-01-02": 2,
				},
				ArticlesSuccess: map[string]int{
					"2020-01-01": 3,
					"2020-01-02": 4,
				},
			},
			"server2": {
				Total: 345678901,
				ArticlesTried: map[string]int{
					"2020-01-02": 6,
					"2020-01-01": 5,
				},
				ArticlesSuccess: map[string]int{
					"2020-01-02": 8,
					"2020-01-01": 7,
				},
			},
		},
	}
	b, err := json.Marshal(statsResponse)
	require.NoError(err)

	var stats ServerStats
	err = json.Unmarshal(b, &stats)
	require.NoError(err)
	require.Equal(123456789, stats.Total)
	require.Equal(2, len(stats.Servers))
	require.Equal(234567890, stats.Servers["server1"].Total)
	require.Equal(2, stats.Servers["server1"].ArticlesTried)
	require.Equal(4, stats.Servers["server1"].ArticlesSuccess)
	require.Equal("2020-01-02", stats.Servers["server1"].DayParsed)
	require.Equal(345678901, stats.Servers["server2"].Total)
	require.Equal(6, stats.Servers["server2"].ArticlesTried)
	require.Equal(8, stats.Servers["server2"].ArticlesSuccess)
	require.Equal("2020-01-02", stats.Servers["server2"].DayParsed)
}
