package collector

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/sabnzbd/auth"
	"github.com/onedr0p/exportarr/internal/sabnzbd/config"
	"github.com/onedr0p/exportarr/internal/sabnzbd/model"
	"golang.org/x/sync/errgroup"
)

// metricPrefix is the namespace prepended to every SABnzbd metric name.
const metricPrefix = "sabnzbd"

// queryDurationBuckets bound the per-endpoint query-duration histograms.
var queryDurationBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// newQueryDurationHistogram builds the duration histogram for one SABnzbd
// endpoint.
func newQueryDurationHistogram(endpoint, url string) prometheus.Histogram {
	return prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:   metricPrefix,
		Name:        endpoint + "_query_duration_seconds",
		Help:        "Distribution of " + endpoint + " endpoint query durations",
		ConstLabels: prometheus.Labels{"url": url},
		Buckets:     queryDurationBuckets,
		// Also expose a sparse native histogram to scrapers that negotiate
		// it; classic buckets above remain for everyone else.
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	})
}

var (
	downloadedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "downloaded_bytes"),
		"Total Bytes Downloaded by SABnzbd",
		[]string{"url"},
		nil,
	)
	serverDownloadedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "server_downloaded_bytes"),
		"Total Bytes Downloaded from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	serverArticlesTotal = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "server_articles_total"),
		"Total Articles Attempted to download from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	serverArticlesSuccess = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "server_articles_success"),
		"Total Articles Successfully downloaded from UseNet Server",
		[]string{"url", "server"},
		nil,
	)
	info = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "info"),
		"Info about the target SabnzbD instance",
		[]string{"url", "version", "status"},
		nil,
	)
	paused = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "paused"),
		"Is the target SabnzbD instance paused",
		[]string{"url"},
		nil,
	)
	pausedAll = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "paused_all"),
		"Are all the target SabnzbD instance's queues paused",
		[]string{"url"},
		nil,
	)
	pauseDuration = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "pause_duration_seconds"),
		"Duration until the SabnzbD instance is unpaused",
		[]string{"url"},
		nil,
	)
	diskUsed = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "disk_used_bytes"),
		"Used Bytes Used on the SabnzbD instance's disk",
		[]string{"url", "folder"},
		nil,
	)
	diskTotal = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "disk_total_bytes"),
		"Total Bytes on the SabnzbD instance's disk",
		[]string{"url", "folder"},
		nil,
	)
	remainingQuota = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "remaining_quota_bytes"),
		"Total Bytes Left in the SabnzbD instance's quota",
		[]string{"url"},
		nil,
	)
	quota = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "quota_bytes"),
		"Total Bytes in the SabnzbD instance's quota",
		[]string{"url"},
		nil,
	)
	cachedArticles = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "article_cache_articles"),
		"Total Articles Cached in the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	cachedBytes = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "article_cache_bytes"),
		"Total Bytes Cached in the SabnzbD instance Article Cache",
		[]string{"url"},
		nil,
	)
	speed = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "speed_bps"),
		"Total Bytes Downloaded per Second by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	speedLimitAbs = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "speed_limit_bps"),
		"Download speed limit of the SabnzbD instance in bytes per second",
		[]string{"url"},
		nil,
	)
	speedLimitPercent = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "speed_limit_percent"),
		"Download speed limit as a percentage of the configured line speed",
		[]string{"url"},
		nil,
	)
	bytesRemaining = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "remaining_bytes"),
		"Total Bytes Remaining to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	bytesTotal = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "total_bytes"),
		"Total Bytes in queue to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	queueLength = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "queue_length"),
		"Total Number of Items in the SabnzbD instance's queue",
		[]string{"url"},
		nil,
	)
	status = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "status"),
		"Status of the SabnzbD instance's queue (0=Unknown, 1=Idle, 2=Paused, 3=Downloading)",
		[]string{"url"},
		nil,
	)
	timeEstimate = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "time_estimate_seconds"),
		"Estimated Time Remaining to Download by the SabnzbD instance",
		[]string{"url"},
		nil,
	)
	warnings = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "queue_warnings"),
		"Total Warnings in the SabnzbD instance's queue",
		[]string{"url"},
		nil,
	)
	collectorError = prometheus.NewDesc(
		prometheus.BuildFQName(metricPrefix, "", "collector_error"),
		"Error while collecting metrics from SABnzbd",
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

// SabnzbdCollector collects SABnzbd metrics over its HTTP API.
type SabnzbdCollector struct {
	cache                    *ServersStatsCache
	client                   *client.Client
	baseURL                  string
	queueQueryDuration       prometheus.Histogram
	serverStatsQueryDuration prometheus.Histogram
}

// NewSabnzbdCollector builds a SabnzbdCollector for the configured SABnzbd
// instance.
// TODO: Add a sab-specific config struct to abstract away the config parsing.
func NewSabnzbdCollector(config *config.SabnzbdConfig) (*SabnzbdCollector, error) {
	author := auth.APIKeyAuth{APIKey: config.APIKey}
	client, err := client.NewClient(config.URL, config.DisableSSLVerify, config.RequestTimeout, author)
	if err != nil {
		return nil, fmt.Errorf("failed to build client: %w", err)
	}

	return &SabnzbdCollector{
		cache:                    NewServersStatsCache(),
		client:                   client,
		baseURL:                  config.URL,
		queueQueryDuration:       newQueryDurationHistogram("queue", config.URL),
		serverStatsQueryDuration: newQueryDurationHistogram("server_stats", config.URL),
	}, nil
}

// getJSON fetches a SABnzbd API mode and decodes the response into T.
func getJSON[T any](s *SabnzbdCollector, mode string, extra ...client.QueryParams) (T, error) {
	params := client.QueryParams{}
	params.Add("mode", mode)
	for _, e := range extra {
		for k, vs := range e {
			for _, v := range vs {
				params.Add(k, v)
			}
		}
	}
	return client.Get[T](s.client, "/api", params)
}

func (s *SabnzbdCollector) getQueueStats() (*model.QueueStats, error) {
	// Slots are never read — keep the payload to the aggregate fields.
	stats, err := getJSON[model.QueueStats](s, "queue", client.QueryParams{"limit": []string{"1"}})
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %w", err)
	}
	return &stats, nil
}

func (s *SabnzbdCollector) getServerStats() (*model.ServerStats, error) {
	stats, err := getJSON[model.ServerStats](s, "server_stats")
	if err != nil {
		return nil, fmt.Errorf("failed to get server stats: %w", err)
	}
	return &stats, nil
}

// Describe implements prometheus.Collector.
func (s *SabnzbdCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collectorError
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
	ch <- speedLimitAbs
	ch <- speedLimitPercent
	ch <- bytesRemaining
	ch <- bytesTotal
	ch <- queueLength
	ch <- status
	ch <- timeEstimate
	ch <- serverDownloadedBytes
	ch <- serverArticlesTotal
	ch <- serverArticlesSuccess
	ch <- warnings
	ch <- s.queueQueryDuration.Desc()
	ch <- s.serverStatsQueryDuration.Desc()
}

// Collect implements prometheus.Collector.
func (s *SabnzbdCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "sabnzbd")
	// A panic in a collector goroutine would crash the whole exporter: degrade
	// to the error gauge instead.
	defer func() {
		if r := recover(); r != nil {
			log.Error("collector panicked", "panic", r)
			ch <- prometheus.MustNewConstMetric(collectorError, prometheus.GaugeValue, 1, s.baseURL)
		}
	}()

	queueStats := &model.QueueStats{}
	serverStats := &model.ServerStats{}

	g := new(errgroup.Group)

	g.Go(func() (err error) {
		// Recover worker panics: this goroutine is outside Collect's recover.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("queue worker panicked: %v", r)
			}
		}()
		qStart := time.Now()
		defer func() {
			s.queueQueryDuration.Observe(time.Since(qStart).Seconds())
			ch <- s.queueQueryDuration
		}()

		queueStats, err = s.getQueueStats()
		if err != nil {
			log.Error("Failed to get queue stats", "error", err)
			return fmt.Errorf("failed to get queue stats: %w", err)
		}
		return nil
	})

	g.Go(func() (err error) {
		// Recover worker panics: this goroutine is outside Collect's recover.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("server_stats worker panicked: %v", r)
			}
		}()
		sStart := time.Now()
		defer func() {
			s.serverStatsQueryDuration.Observe(time.Since(sStart).Seconds())
			ch <- s.serverStatsQueryDuration
		}()

		serverStats, err = s.getServerStats()
		if err != nil {
			log.Error("Failed to get server stats", "error", err)
			return fmt.Errorf("failed to get server stats: %w", err)
		}

		return s.cache.Update(*serverStats)
	})

	if err := g.Wait(); err != nil {
		log.Error("Failed to get stats", "error", err)
		ch <- prometheus.MustNewConstMetric(collectorError, prometheus.GaugeValue, 1, s.baseURL)

		return
	}

	ch <- prometheus.MustNewConstMetric(
		downloadedBytes, prometheus.CounterValue, float64(s.cache.GetTotal()), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		info, prometheus.GaugeValue, 1, s.baseURL, queueStats.Version, queueStats.Status.String(),
	)
	ch <- prometheus.MustNewConstMetric(
		paused, prometheus.GaugeValue, boolToFloat(queueStats.Paused), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		pausedAll, prometheus.GaugeValue, boolToFloat(queueStats.PausedAll), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		pauseDuration, prometheus.GaugeValue, queueStats.PauseDuration.Seconds(), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		diskUsed, prometheus.GaugeValue, queueStats.DownloadDirDiskspaceUsed, s.baseURL, "download",
	)
	ch <- prometheus.MustNewConstMetric(
		diskUsed, prometheus.GaugeValue, queueStats.CompletedDirDiskspaceUsed, s.baseURL, "complete",
	)
	ch <- prometheus.MustNewConstMetric(
		diskTotal, prometheus.GaugeValue, queueStats.DownloadDirDiskspaceTotal, s.baseURL, "download",
	)
	ch <- prometheus.MustNewConstMetric(
		diskTotal, prometheus.GaugeValue, queueStats.CompletedDirDiskspaceTotal, s.baseURL, "complete",
	)
	ch <- prometheus.MustNewConstMetric(
		remainingQuota, prometheus.GaugeValue, queueStats.RemainingQuota, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		quota, prometheus.GaugeValue, queueStats.Quota, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		cachedArticles, prometheus.GaugeValue, queueStats.CacheArt, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		cachedBytes, prometheus.GaugeValue, queueStats.CacheSize, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		speed, prometheus.GaugeValue, queueStats.Speed, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		speedLimitAbs, prometheus.GaugeValue, queueStats.SpeedLimitAbs, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		speedLimitPercent, prometheus.GaugeValue, queueStats.SpeedLimit, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		bytesRemaining, prometheus.GaugeValue, queueStats.RemainingSize, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		bytesTotal, prometheus.GaugeValue, queueStats.Size, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		queueLength, prometheus.GaugeValue, queueStats.ItemsInQueue, s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		status, prometheus.GaugeValue, queueStats.Status.Float64(), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		timeEstimate, prometheus.GaugeValue, queueStats.TimeEstimate.Seconds(), s.baseURL,
	)
	ch <- prometheus.MustNewConstMetric(
		warnings, prometheus.GaugeValue, queueStats.HaveWarnings, s.baseURL,
	)

	for name, stats := range s.cache.GetServerMap() {
		ch <- prometheus.MustNewConstMetric(
			serverDownloadedBytes, prometheus.CounterValue, float64(stats.GetTotal()), s.baseURL, name,
		)
		ch <- prometheus.MustNewConstMetric(
			serverArticlesTotal, prometheus.CounterValue, float64(stats.GetArticlesTried()), s.baseURL, name,
		)
		ch <- prometheus.MustNewConstMetric(
			serverArticlesSuccess, prometheus.CounterValue, float64(stats.GetArticlesSuccess()), s.baseURL, name,
		)
	}
}
