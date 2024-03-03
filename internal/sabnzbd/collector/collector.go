package collector

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/sabnzbd/auth"
	"github.com/onedr0p/exportarr/internal/sabnzbd/config"
	"github.com/onedr0p/exportarr/internal/sabnzbd/model"
	"golang.org/x/sync/errgroup"
)

var METRIC_PREFIX = "sabnzbd"

var (
	downloadedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "downloaded_bytes"),
		"Total Bytes Downloaded by SABnzbd",
		[]string{"url"},
		nil,
	)
	serverDownloadedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "server_downloaded_bytes"),
		"Total Bytes Downloaded from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	serverArticlesTotal = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "server_articles_total"),
		"Total Articles Attempted to download from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	serverArticlesSuccess = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "server_articles_success"),
		"Total Articles Successfully downloaded from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	info = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "info"),
		"Info about the target SabnzbD instance",
		[]string{"url", "version", "status"},
		nil,
	)
	paused = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "paused"),
		"Is the target SabnzbD instance paused",
		[]string{"url"},
		nil,
	)
	pausedAll = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "paused_all"),
		"Are all the target SabnzbD instance's queues paused",
		[]string{"url"},
		nil,
	)
	pauseDuration = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "pause_duration_seconds"),
		"Duration until the SabnzbD instance is unpaused",
		[]string{"url"},
		nil,
	)
	diskUsed = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "disk_used_bytes"),
		"Used Bytes Used on the SabnzbD instance's disk",
		[]string{"url", "folder"},
		nil,
	)
	diskTotal = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "disk_total_bytes"),
		"Total Bytes on the SabnzbD instance's disk",
		[]string{"url", "folder"},
		nil,
	)
	remainingQuota = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "remaining_quota_bytes"),
		"Total Bytes Left in the SabnzbD instance's quota",
		[]string{"url"},
		nil,
	)
	quota = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "quota_bytes"),
		"Total Bytes in the SabnzbD instance's quota",
		[]string{"url"},
		nil,
	)
	cachedArticles = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "article_cache_articles"),
		"Total Articles Cached in the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	cachedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "article_cache_bytes"),
		"Total Bytes Cached in the SabnzbD instance Article Cache",
		[]string{"url"},
		nil,
	)
	speed = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "speed_bps"),
		"Total Bytes Downloaded per Second by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	bytesRemaining = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "remaining_bytes"),
		"Total Bytes Remaining to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	bytesTotal = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "total_bytes"),
		"Total Bytes in queue to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	queueLength = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "queue_length"),
		"Total Number of Items in the SabnzbD instance's queue",
		[]string{"url"},
		nil,
	)
	status = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "status"),
		"Status of the SabnzbD instance's queue (0=Unknown, 1=Idle, 2=Paused, 3=Downloading)",
		[]string{"url"},
		nil,
	)
	timeEstimate = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "time_estimate_seconds"),
		"Estimated Time Remaining to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	warnings = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "queue_warnings"),
		"Total Warnings in the SabnzbD instance's queue",
		[]string{"url"},
		nil,
	)
	queueQueryDuration = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "queue_query_duration_seconds"),
		"Duration querying the queue endpoint of SabnzbD",
		[]string{"url"},
		nil,
	)
	serverStatsQueryDuration = prometheus.NewDesc(
		prometheus.BuildFQName(METRIC_PREFIX, "", "server_stats_query_duration_seconds"),
		"Duration querying the server_stats endpoint of SabnzbD",
		[]string{"url"},
		nil,
	)
)

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}

	return 0
}

type SabnzbdCollector struct {
	cache   *ServersStatsCache
	client  *client.Client
	baseURL string
}

// TODO: Add a sab-specific config struct to abstract away the config parsing
func NewSabnzbdCollector(config *config.SabnzbdConfig) (*SabnzbdCollector, error) {
	auther := auth.ApiKeyAuth{ApiKey: config.ApiKey}
	client, err := client.NewClient(config.URL, config.DisableSSLVerify, auther)
	if err != nil {
		return nil, fmt.Errorf("Failed to build client: %w", err)
	}

	return &SabnzbdCollector{
		cache:   NewServersStatsCache(),
		client:  client,
		baseURL: config.URL,
	}, nil
}

func (s *SabnzbdCollector) doRequest(mode string, target interface{}) error {
	params := client.QueryParams{}
	params.Add("mode", mode)
	return s.client.DoRequest("/api", target, params)
}

func (s *SabnzbdCollector) getQueueStats() (*model.QueueStats, error) {
	var stats = &model.QueueStats{}

	err := s.doRequest("queue", stats)
	if err != nil {
		return nil, fmt.Errorf("Failed to get queue stats: %w", err)
	}

	return stats, nil
}

func (s *SabnzbdCollector) getServerStats() (*model.ServerStats, error) {
	var stats = &model.ServerStats{}
	err := s.doRequest("server_stats", stats)
	if err != nil {
		return nil, fmt.Errorf("Failed to get server stats: %w", err)
	}
	return stats, nil
}

func (e *SabnzbdCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- downloadedBytes
	ch <- info
	ch <- paused
	ch <- pausedAll
	ch <- pauseDuration
	ch <- diskUsed
	ch <- diskTotal
	ch <- remainingQuota
	ch <- quota
	ch <- cachedArticles
	ch <- cachedBytes
	ch <- speed
	ch <- bytesRemaining
	ch <- bytesTotal
	ch <- queueLength
	ch <- status
	ch <- timeEstimate
	ch <- serverDownloadedBytes
	ch <- serverArticlesTotal
	ch <- serverArticlesSuccess
	ch <- warnings
	ch <- queueQueryDuration
	ch <- serverStatsQueryDuration
}

func (e *SabnzbdCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "sabnzbd")

	queueStats := &model.QueueStats{}
	serverStats := &model.ServerStats{}

	g := new(errgroup.Group)

	g.Go(func() error {
		qStart := time.Now()
		defer func() { //nolint:wsl
			ch <- prometheus.MustNewConstMetric(
				queueQueryDuration, prometheus.GaugeValue, time.Since(qStart).Seconds(), e.baseURL)
		}()

		var err error
		queueStats, err = e.getQueueStats()
		if err != nil {
			log.Errorw("Failed to get queue stats", "error", err)
			return fmt.Errorf("failed to get queue stats: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		sStart := time.Now()
		defer func() { //nolint:wsl
			ch <- prometheus.MustNewConstMetric(
				serverStatsQueryDuration, prometheus.GaugeValue, time.Since(sStart).Seconds(), e.baseURL)
		}()

		var err error
		serverStats, err = e.getServerStats()
		if err != nil {
			log.Errorw("Failed to get server stats", "error", err)
			return fmt.Errorf("failed to get server stats: %w", err)
		}

		return e.cache.Update(*serverStats)
	})

	if err := g.Wait(); err != nil {
		log.Errorw("Failed to get stats", "error", err)
		ch <- prometheus.NewInvalidMetric(
			prometheus.NewDesc("sabnzbd_collector_error", "Error getting stats", nil, prometheus.Labels{"target": e.baseURL}),
			err,
		)

		return
	}

	ch <- prometheus.MustNewConstMetric(
		downloadedBytes, prometheus.CounterValue, float64(e.cache.GetTotal()), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		info, prometheus.GaugeValue, 1, e.baseURL, queueStats.Version, queueStats.Status.String(),
	)
	ch <- prometheus.MustNewConstMetric(
		paused, prometheus.GaugeValue, boolToFloat(queueStats.Paused), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		pausedAll, prometheus.GaugeValue, boolToFloat(queueStats.PausedAll), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		pauseDuration, prometheus.GaugeValue, queueStats.PauseDuration.Seconds(), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		diskUsed, prometheus.GaugeValue, queueStats.DownloadDirDiskspaceUsed, e.baseURL, "download",
	)
	ch <- prometheus.MustNewConstMetric(
		diskUsed, prometheus.GaugeValue, queueStats.CompletedDirDiskspaceUsed, e.baseURL, "complete",
	)
	ch <- prometheus.MustNewConstMetric(
		diskTotal, prometheus.GaugeValue, queueStats.DownloadDirDiskspaceTotal, e.baseURL, "download",
	)
	ch <- prometheus.MustNewConstMetric(
		diskTotal, prometheus.GaugeValue, queueStats.CompletedDirDiskspaceTotal, e.baseURL, "complete",
	)
	ch <- prometheus.MustNewConstMetric(
		remainingQuota, prometheus.GaugeValue, queueStats.RemainingQuota, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		quota, prometheus.GaugeValue, queueStats.Quota, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		cachedArticles, prometheus.GaugeValue, queueStats.CacheArt, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		cachedBytes, prometheus.GaugeValue, queueStats.CacheSize, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		speed, prometheus.GaugeValue, queueStats.Speed, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		bytesRemaining, prometheus.GaugeValue, queueStats.RemainingSize, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		bytesTotal, prometheus.GaugeValue, queueStats.Size, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		queueLength, prometheus.GaugeValue, queueStats.ItemsInQueue, e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		status, prometheus.GaugeValue, queueStats.Status.Float64(), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		timeEstimate, prometheus.GaugeValue, queueStats.TimeEstimate.Seconds(), e.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		warnings, prometheus.GaugeValue, queueStats.HaveWarnings, e.baseURL,
	)

	for name, stats := range e.cache.GetServerMap() {
		ch <- prometheus.MustNewConstMetric(
			serverDownloadedBytes, prometheus.CounterValue, float64(stats.GetTotal()), e.baseURL, name,
		)
		ch <- prometheus.MustNewConstMetric(
			serverArticlesTotal, prometheus.CounterValue, float64(stats.GetArticlesTried()), e.baseURL, name,
		)
		ch <- prometheus.MustNewConstMetric(
			serverArticlesSuccess, prometheus.CounterValue, float64(stats.GetArticlesSuccess()), e.baseURL, name,
		)
	}
}
