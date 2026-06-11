package model

import (
	"encoding/json"
	"fmt"
	"github.com/onedr0p/exportarr/internal/assert"
	"os"
	"testing"
	"time"
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
	assert.Equal(t, DOWNLOADING.String(), "Downloading")
	assert.Equal(t, PAUSED.String(), "Paused")
	assert.Equal(t, IDLE.String(), "Idle")
	assert.Equal(t, Status(999).String(), "Unknown")
}

func TestStatusFromString(t *testing.T) {
	assert.Equal(t, StatusFromString("Downloading"), DOWNLOADING)
	assert.Equal(t, StatusFromString("Paused"), PAUSED)
	assert.Equal(t, StatusFromString("Idle"), IDLE)
	assert.Equal(t, StatusFromString("Unknown"), UNKNOWN)
	assert.Equal(t, StatusFromString("Unknown"), UNKNOWN)
}

func TestStatusToFloat(t *testing.T) {
	assert.Equal(t, DOWNLOADING.Float64(), 3.0)
	assert.Equal(t, PAUSED.Float64(), 2.0)
	assert.Equal(t, IDLE.Float64(), 1.0)
	assert.Equal(t, UNKNOWN.Float64(), 0.0)
}

func TestQueueStats_UnmarshalJSON(t *testing.T) {
	queue, err := os.ReadFile("../testdata/queue.json")
	assert.NoError(t, err)

	var queueStats QueueStats
	err = queueStats.UnmarshalJSON(queue)
	assert.NoError(t, err)
	assert.Equal(t, queueStats.Version, "3.7.2")
	assert.False(t, queueStats.Paused)
	assert.False(t, queueStats.PausedAll)
	assert.Equal(t, queueStats.PauseDuration, time.Duration(0))
	assert.Equal(t, queueStats.DownloadDirDiskspaceUsed, 8.712770656665602e+12)
	assert.Equal(t, queueStats.DownloadDirDiskspaceTotal, 4.6050639347712e+13)
	assert.Equal(t, queueStats.CompletedDirDiskspaceUsed, 8.771826456985602e+12)
	assert.Equal(t, queueStats.CompletedDirDiskspaceTotal, 4.6051713089536e+13)
	assert.Equal(t, queueStats.SpeedLimit, 100.0)
	assert.Equal(t, queueStats.SpeedLimitAbs, 1.048576e+09)
	assert.Equal(t, queueStats.HaveWarnings, 0.0)
	assert.Equal(t, queueStats.Quota, 1.07911053312e+12)
	assert.True(t, queueStats.HaveQuota)
	assert.Equal(t, queueStats.RemainingQuota, 1.073741824e+12)
	assert.Equal(t, queueStats.CacheArt, 0.0)
	assert.Equal(t, queueStats.CacheSize, 0.0)
	assert.Equal(t, queueStats.Speed, 358.4)
	assert.Equal(t, queueStats.RemainingSize, 3.21070825472e+09)
	assert.Equal(t, queueStats.Size, 3.21175683072e+09)
	assert.Equal(t, queueStats.ItemsInQueue, 2.0)
	assert.Equal(t, queueStats.Status, DOWNLOADING)
	d, _ := time.ParseDuration("2495h59m3s")
	assert.Equal(t, queueStats.TimeEstimate, d)

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

	for _, parameter := range parameters {
		statsResponse := fmt.Sprintf(`{"queue": {"left_quota": "%s"}}`, parameter.input)
		var stats QueueStats
		err := json.Unmarshal([]byte(statsResponse), &stats)
		assert.NoError(t, err)
		assert.Equal(t, stats.RemainingQuota, parameter.expected)
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

	for _, parameter := range parameters {
		var stats QueueStats
		statsResponse := fmt.Sprintf(`{ "queue": {"timeleft": "%s"}}`, parameter.input)
		err := json.Unmarshal([]byte(statsResponse), &stats)
		assert.NoError(t, err)
		assert.Equal(t, stats.TimeEstimate, parameter.expected)
	}
}

func TestServerStats_UnmarshalJSON(t *testing.T) {

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
	assert.NoError(t, err)

	var stats ServerStats
	err = json.Unmarshal(b, &stats)
	assert.NoError(t, err)
	assert.Equal(t, stats.Total, 123456789)
	assert.Equal(t, len(stats.Servers), 2)
	assert.Equal(t, stats.Servers["server1"].Total, 234567890)
	assert.Equal(t, stats.Servers["server1"].ArticlesTried, 2)
	assert.Equal(t, stats.Servers["server1"].ArticlesSuccess, 4)
	assert.Equal(t, stats.Servers["server1"].DayParsed, "2020-01-02")
	assert.Equal(t, stats.Servers["server2"].Total, 345678901)
	assert.Equal(t, stats.Servers["server2"].ArticlesTried, 6)
	assert.Equal(t, stats.Servers["server2"].ArticlesSuccess, 8)
	assert.Equal(t, stats.Servers["server2"].DayParsed, "2020-01-02")
}
