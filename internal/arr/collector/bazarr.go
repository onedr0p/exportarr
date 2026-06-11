// Package collector implements the per-app *arr Prometheus collectors.
package collector

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type stats struct {
	downloaded  int
	monitored   int
	unmonitored int
	missing     int
	wanted      int
	fileSize    int64
	history     int
	languages   map[string]int
	scores      map[string]int
	providers   map[string]int
	mut         sync.Mutex
}

func newStats() *stats {
	return &stats{
		languages: make(map[string]int),
		scores:    make(map[string]int),
		providers: make(map[string]int),
	}
}

type bazarrCollector struct {
	collectMu                  sync.Mutex // Guards against overlapping collections (#380)
	client                     *client.Client
	config                     *config.ArrConfig // App configuration
	subtitlesHistoryMetric     *prometheus.Desc  // Total number of subtitles history
	subtitlesDownloadedMetric  *prometheus.Desc  // Total number of subtitles downloaded
	subtitlesMonitoredMetric   *prometheus.Desc  // Total number of subtitles monitored
	subtitlesUnmonitoredMetric *prometheus.Desc  // Total number of subtitles unmonitored
	subtitlesWantedMetric      *prometheus.Desc  // Total number of wanted subtitle
	subtitlesMissingMetric     *prometheus.Desc  // Total number of missing subtitle
	subtitlesFileSizeMetric    *prometheus.Desc  // Total fizesize of all subtitle in bytes

	episodeSubtitlesHistoryMetric     *prometheus.Desc // Total number of episode subtitles history
	episodeSubtitlesDownloadedMetric  *prometheus.Desc // Total number of episode subtitles downloaded
	episodeSubtitlesMonitoredMetric   *prometheus.Desc // Total number of episode subtitles monitored
	episodeSubtitlesUnmonitoredMetric *prometheus.Desc // Total number of episode subtitles unmonitored
	episodeSubtitlesWantedMetric      *prometheus.Desc // Total number of episode wanted subtitle
	episodeSubtitlesMissingMetric     *prometheus.Desc // Total number of episode missing subtitle
	episodeSubtitlesFileSizeMetric    *prometheus.Desc // Total fizesize of all episode subtitle in bytes

	movieSubtitlesHistoryMetric     *prometheus.Desc // Total number of movie subtitles history
	movieSubtitlesDownloadedMetric  *prometheus.Desc // Total number of movie subtitles downloaded
	movieSubtitlesMonitoredMetric   *prometheus.Desc // Total number of movie subtitles monitored
	movieSubtitlesUnmonitoredMetric *prometheus.Desc // Total number of movie subtitles unmonitored
	movieSubtitlesWantedMetric      *prometheus.Desc // Total number of movie wanted subtitle
	movieSubtitlesMissingMetric     *prometheus.Desc // Total number of movie missing subtitle
	movieSubtitlesFileSizeMetric    *prometheus.Desc // Total fizesize of all movie subtitle in bytes

	subtitlesLanguageMetric *prometheus.Desc // Total number of subtitle by language
	subtitlesScoreMetric    *prometheus.Desc // Total number of subtitle by score
	subtitlesProviderMetric *prometheus.Desc // Total number of subtitle by provider

	systemHealthMetric       *prometheus.Desc // Total number of health issues
	systemStatusMetric       *prometheus.Desc // Total number of system statuses
	throttledProvidersMetric *prometheus.Desc // Number of throttled subtitle providers
	signalrConnectedMetric   *prometheus.Desc // Upstream SignalR connection state
	errorMetric              *prometheus.Desc // Error Description for use with InvalidMetric
}

// NewBazarrCollector builds a collector for bazarr subtitle statistics.
func NewBazarrCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	subtitleText := "subtitles"
	episodeText := "episode"
	movieText := "movie"
	return &bazarrCollector{
		client:                            httpClient,
		config:                            c,
		subtitlesHistoryMetric:            newDesc(c.App, fmt.Sprintf("%s_history_total", subtitleText), fmt.Sprintf("Total number of history %s", subtitleText), nil, c.URL),
		subtitlesDownloadedMetric:         newDesc(c.App, fmt.Sprintf("%s_downloaded_total", subtitleText), fmt.Sprintf("Total number of downloaded %s", subtitleText), nil, c.URL),
		subtitlesMonitoredMetric:          newDesc(c.App, fmt.Sprintf("%s_monitored_total", subtitleText), fmt.Sprintf("Total number of monitored %s", subtitleText), nil, c.URL),
		subtitlesUnmonitoredMetric:        newDesc(c.App, fmt.Sprintf("%s_unmonitored_total", subtitleText), fmt.Sprintf("Total number of unmonitored %s", subtitleText), nil, c.URL),
		subtitlesWantedMetric:             newDesc(c.App, fmt.Sprintf("%s_wanted_total", subtitleText), fmt.Sprintf("Total number of wanted %s", subtitleText), nil, c.URL),
		subtitlesMissingMetric:            newDesc(c.App, fmt.Sprintf("%s_missing_total", subtitleText), fmt.Sprintf("Total number of missing %s", subtitleText), nil, c.URL),
		subtitlesFileSizeMetric:           newDesc(c.App, fmt.Sprintf("%s_filesize_total", subtitleText), fmt.Sprintf("Total filesize of all %s", subtitleText), nil, c.URL),
		episodeSubtitlesHistoryMetric:     newDesc(c.App, fmt.Sprintf("%s_%s_history_total", episodeText, subtitleText), fmt.Sprintf("Total number of history %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesDownloadedMetric:  newDesc(c.App, fmt.Sprintf("%s_%s_downloaded_total", episodeText, subtitleText), fmt.Sprintf("Total number of downloaded %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesMonitoredMetric:   newDesc(c.App, fmt.Sprintf("%s_%s_monitored_total", episodeText, subtitleText), fmt.Sprintf("Total number of monitored %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesUnmonitoredMetric: newDesc(c.App, fmt.Sprintf("%s_%s_unmonitored_total", episodeText, subtitleText), fmt.Sprintf("Total number of unmonitored %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesWantedMetric:      newDesc(c.App, fmt.Sprintf("%s_%s_wanted_total", episodeText, subtitleText), fmt.Sprintf("Total number of wanted %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesMissingMetric:     newDesc(c.App, fmt.Sprintf("%s_%s_missing_total", episodeText, subtitleText), fmt.Sprintf("Total number of missing %s %s", episodeText, subtitleText), nil, c.URL),
		episodeSubtitlesFileSizeMetric:    newDesc(c.App, fmt.Sprintf("%s_%s_filesize_total", episodeText, subtitleText), fmt.Sprintf("Total filesize of all %s %s", episodeText, subtitleText), nil, c.URL),
		movieSubtitlesHistoryMetric:       newDesc(c.App, fmt.Sprintf("%s_%s_history_total", movieText, subtitleText), fmt.Sprintf("Total number of history %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesDownloadedMetric:    newDesc(c.App, fmt.Sprintf("%s_%s_downloaded_total", movieText, subtitleText), fmt.Sprintf("Total number of downloaded %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesMonitoredMetric:     newDesc(c.App, fmt.Sprintf("%s_%s_monitored_total", movieText, subtitleText), fmt.Sprintf("Total number of monitored %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesUnmonitoredMetric:   newDesc(c.App, fmt.Sprintf("%s_%s_unmonitored_total", movieText, subtitleText), fmt.Sprintf("Total number of unmonitored %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesWantedMetric:        newDesc(c.App, fmt.Sprintf("%s_%s_wanted_total", movieText, subtitleText), fmt.Sprintf("Total number of wanted %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesMissingMetric:       newDesc(c.App, fmt.Sprintf("%s_%s_missing_total", movieText, subtitleText), fmt.Sprintf("Total number of missing %s %s", movieText, subtitleText), nil, c.URL),
		movieSubtitlesFileSizeMetric:      newDesc(c.App, fmt.Sprintf("%s_%s_filesize_total", movieText, subtitleText), fmt.Sprintf("Total filesize of all %s %s", movieText, subtitleText), nil, c.URL),
		subtitlesLanguageMetric:           newDesc(c.App, fmt.Sprintf("%s_language_total", subtitleText), fmt.Sprintf("Total number of downloaded %s by language", subtitleText), []string{"language"}, c.URL),
		subtitlesScoreMetric:              newDesc(c.App, fmt.Sprintf("%s_score", subtitleText), fmt.Sprintf("Distribution of downloaded %s scores (percent)", subtitleText), nil, c.URL),
		subtitlesProviderMetric:           newDesc(c.App, fmt.Sprintf("%s_provider_total", subtitleText), fmt.Sprintf("Total number of downloaded %s by provider", subtitleText), []string{"provider"}, c.URL),
		systemHealthMetric:                newDesc(c.App, "system_health_issues", "Total number of health issues by object and issue", []string{"object", "issue"}, c.URL),
		systemStatusMetric:                newDesc(c.App, "system_status", "System Status", nil, c.URL),
		throttledProvidersMetric: newDesc(c.App, "throttled_providers",
			"Number of currently throttled subtitle providers", nil, c.URL),
		signalrConnectedMetric: newDesc(c.App, "signalr_connected",
			"Whether bazarr's SignalR connection to the upstream app is live (1) or not (0)", []string{"app"}, c.URL),
		errorMetric: newDesc(c.App, "collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *bazarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.subtitlesHistoryMetric
	ch <- collector.subtitlesDownloadedMetric
	ch <- collector.subtitlesMonitoredMetric
	ch <- collector.subtitlesUnmonitoredMetric
	ch <- collector.subtitlesWantedMetric
	ch <- collector.subtitlesMissingMetric
	ch <- collector.subtitlesFileSizeMetric

	ch <- collector.episodeSubtitlesHistoryMetric
	ch <- collector.episodeSubtitlesDownloadedMetric
	ch <- collector.episodeSubtitlesMonitoredMetric
	ch <- collector.episodeSubtitlesUnmonitoredMetric
	ch <- collector.episodeSubtitlesWantedMetric
	ch <- collector.episodeSubtitlesMissingMetric
	ch <- collector.episodeSubtitlesFileSizeMetric

	ch <- collector.movieSubtitlesHistoryMetric
	ch <- collector.movieSubtitlesDownloadedMetric
	ch <- collector.movieSubtitlesMonitoredMetric
	ch <- collector.movieSubtitlesUnmonitoredMetric
	ch <- collector.movieSubtitlesWantedMetric
	ch <- collector.movieSubtitlesMissingMetric
	ch <- collector.movieSubtitlesFileSizeMetric
	ch <- collector.movieSubtitlesFileSizeMetric

	ch <- collector.subtitlesScoreMetric
	ch <- collector.subtitlesLanguageMetric
	ch <- collector.subtitlesProviderMetric

	ch <- collector.systemStatusMetric
	ch <- collector.systemHealthMetric
	ch <- collector.throttledProvidersMetric
	ch <- collector.signalrConnectedMetric
}

func (collector *bazarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "bazarr")
	defer recoverCollect(log, ch, collector.errorMetric)
	// If a previous collection is still running (slow target, overlapping
	// scrapes), skip this one instead of stacking more load onto the app —
	// overlapping walks are how a slow instance ends up pinned at 100% CPU
	// (https://github.com/onedr0p/exportarr/issues/380).
	if !collector.collectMu.TryLock() {
		log.Warn("previous collection still in progress; skipping this scrape")
		ch <- prometheus.MustNewConstMetric(collector.errorMetric, prometheus.GaugeValue, 1)
		return
	}
	defer collector.collectMu.Unlock()
	c := collector.client
	tseries := time.Now()

	// Badges is one cheap call carrying the missing-episode count, throttled
	// providers, and upstream connection state — fetch it first and share it
	// (https://github.com/onedr0p/exportarr/issues/407).
	var badges *model.BazarrBadges
	if b, err := client.Get[model.BazarrBadges](c, "badges"); err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting badges", "error", err)
	} else {
		badges = &b
	}

	// The subtitle walks and system endpoints are independent: overlap them.
	var wg sync.WaitGroup
	wg.Go(func() {
		defer recoverCollect(log, ch, collector.errorMetric)
		collector.episodeMovieMetrics(ch, c, badges)
	})
	wg.Go(func() {
		defer recoverCollect(log, ch, collector.errorMetric)
		collector.systemMetrics(ch, c, badges)
	})
	wg.Wait()

	mt := time.Since(tseries)
	log.Debug("All Completed", "duration", mt)
}

func (collector *bazarrCollector) episodeMovieMetrics(ch chan<- prometheus.Metric, c *client.Client, badges *model.BazarrBadges) {
	// Episode and movie walks are independent: collect them concurrently.
	episodeStats := newStats()
	var movieStats *stats
	var wg sync.WaitGroup
	if !collector.config.DisableEpisodeMetrics {
		wg.Go(func() {
			defer recoverCollect(slog.With("collector", "bazarr"), ch, collector.errorMetric)
			episodeStats = collector.collectEpisodeStats(ch, c)
		})
	} else if badges != nil {
		// Without the per-episode walk, the badges endpoint still provides the
		// missing-episode count for one cheap call — and keeps the combined
		// totals honest (#407).
		episodeStats.missing = badges.Episodes
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesMissingMetric, prometheus.GaugeValue, float64(badges.Episodes))
	}
	wg.Go(func() {
		defer recoverCollect(slog.With("collector", "bazarr"), ch, collector.errorMetric)
		movieStats = collector.collectMovieStats(ch, c)
	})
	wg.Wait()
	if episodeStats == nil || movieStats == nil {
		return
	}

	if !collector.config.DisableEpisodeMetrics {
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesHistoryMetric, prometheus.GaugeValue, float64(episodeStats.history))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesDownloadedMetric, prometheus.GaugeValue, float64(episodeStats.downloaded))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesMonitoredMetric, prometheus.GaugeValue, float64(episodeStats.monitored))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesUnmonitoredMetric, prometheus.GaugeValue, float64(episodeStats.unmonitored))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesWantedMetric, prometheus.GaugeValue, float64(episodeStats.wanted))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesMissingMetric, prometheus.GaugeValue, float64(episodeStats.missing))
		ch <- prometheus.MustNewConstMetric(collector.episodeSubtitlesFileSizeMetric, prometheus.GaugeValue, float64(episodeStats.fileSize))
	}

	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesHistoryMetric, prometheus.GaugeValue, float64(movieStats.history))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesDownloadedMetric, prometheus.GaugeValue, float64(movieStats.downloaded))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesMonitoredMetric, prometheus.GaugeValue, float64(movieStats.monitored))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesUnmonitoredMetric, prometheus.GaugeValue, float64(movieStats.unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesWantedMetric, prometheus.GaugeValue, float64(movieStats.wanted))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesMissingMetric, prometheus.GaugeValue, float64(movieStats.missing))
	ch <- prometheus.MustNewConstMetric(collector.movieSubtitlesFileSizeMetric, prometheus.GaugeValue, float64(movieStats.fileSize))

	ch <- prometheus.MustNewConstMetric(collector.subtitlesDownloadedMetric, prometheus.GaugeValue, float64(episodeStats.downloaded)+float64(movieStats.downloaded))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesMonitoredMetric, prometheus.GaugeValue, float64(episodeStats.monitored)+float64(movieStats.monitored))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesUnmonitoredMetric, prometheus.GaugeValue, float64(episodeStats.unmonitored)+float64(movieStats.unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesWantedMetric, prometheus.GaugeValue, float64(episodeStats.wanted)+float64(movieStats.wanted))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesMissingMetric, prometheus.GaugeValue, float64(episodeStats.missing)+float64(movieStats.missing))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesFileSizeMetric, prometheus.GaugeValue, float64(episodeStats.fileSize)+float64(movieStats.fileSize))
	ch <- prometheus.MustNewConstMetric(collector.subtitlesHistoryMetric, prometheus.GaugeValue, float64(episodeStats.history)+float64(movieStats.history))

	// Merge movie counts into the episode maps, then emit one metric per label.
	emitMergedCounts(ch, collector.subtitlesLanguageMetric, episodeStats.languages, movieStats.languages)
	emitScoreHistogram(ch, collector.subtitlesScoreMetric, episodeStats.scores, movieStats.scores)
	emitMergedCounts(ch, collector.subtitlesProviderMetric, episodeStats.providers, movieStats.providers)
}

// scoreBuckets are the inclusive upper bounds of the subtitle-score histogram;
// subtitle scores cluster high, so resolution tightens toward 100%.
var scoreBuckets = []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 100}

// emitScoreHistogram merges score counts and emits them as a histogram.
// Scores are percentages with two decimals: as label values they would create
// unbounded cardinality (https://github.com/onedr0p/exportarr/issues/239).
func emitScoreHistogram(ch chan<- prometheus.Metric, desc *prometheus.Desc, dst, src map[string]int) {
	for k, c := range src {
		dst[k] += c
	}
	keys := slices.Sorted(maps.Keys(dst)) // deterministic float accumulation
	var count uint64
	var sum float64
	buckets := make(map[float64]uint64, len(scoreBuckets))
	for _, ub := range scoreBuckets {
		buckets[ub] = 0 // emit every bucket, even when empty
	}
	for _, raw := range keys {
		score, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(raw), "%"), 64)
		if err != nil {
			slog.Debug("Skipping unparseable subtitle score", "score", raw)
			continue
		}
		n := uint64(dst[raw]) //nolint:gosec // history counts are small and non-negative
		count += n
		sum += score * float64(dst[raw])
		for _, ub := range scoreBuckets {
			if score <= ub {
				buckets[ub] += n
			}
		}
	}
	ch <- prometheus.MustNewConstHistogram(desc, count, sum, buckets)
}

// emitMergedCounts merges src counts into dst, then emits dst as one gauge per
// label value.
func emitMergedCounts(ch chan<- prometheus.Metric, desc *prometheus.Desc, dst, src map[string]int) {
	for k, count := range src {
		dst[k] += count
	}
	for k, count := range dst {
		ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue, float64(count), k)
	}
}

func (collector *bazarrCollector) collectEpisodeStats(ch chan<- prometheus.Metric, c *client.Client) *stats {
	log := slog.With("collector", "bazarr")
	episodeStats := newStats()

	mseries := time.Now()

	series, err := client.Get[model.BazarrSeries](c, "series")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting series", "error", err)
		return nil
	}

	ids := []string{}
	for _, s := range series.Data {
		ids = append(ids, strconv.Itoa(s.ID))
	}
	eg := errgroup.Group{}
	eg.SetLimit(collector.config.Bazarr.SeriesBatchConcurrency)
	// slices.Chunk yields nothing for an empty id list, so an empty bazarr
	// never hits the episodes endpoint (#244). max guards a zero batch size.
	for batch := range slices.Chunk(ids, max(1, collector.config.Bazarr.SeriesBatchSize)) {
		params := client.QueryParams{"seriesid[]": batch}
		goRecoverable(&eg, func() error {
			episodes, err := client.Get[model.BazarrEpisodes](c, "episodes", params)
			if err != nil {
				return err
			}
			episodeStats.mut.Lock()
			defer episodeStats.mut.Unlock()
			for _, e := range episodes.Data {
				if !e.Monitored {
					episodeStats.unmonitored++
				} else {
					episodeStats.monitored++
				}

				if len(e.MissingSubtitles) > 0 {
					episodeStats.missing += len(e.MissingSubtitles)
					if len(e.Subtitles) == 0 {
						episodeStats.wanted++
					}
				}

				if len(e.Subtitles) > 0 {
					episodeStats.downloaded += len(e.Subtitles)
					for _, subtitle := range e.Subtitles {
						if subtitle.Language != "" {
							episodeStats.languages[subtitle.Language]++
						} else if subtitle.Code2 != "" {
							episodeStats.languages[subtitle.Code2]++
						} else if subtitle.Code3 != "" {
							episodeStats.languages[subtitle.Code3]++
						}

						if subtitle.Size != 0 {
							episodeStats.fileSize += subtitle.Size
						}
					}
				}
			}
			return nil
		})
	}
	// Episode history is independent of the per-series walk: fetch it on the
	// same group so the two overlap.
	goRecoverable(&eg, func() error {
		history, err := client.Get[model.BazarrHistory](c, "episodes/history")
		if err != nil {
			return fmt.Errorf("getting episodes history: %w", err)
		}
		episodeStats.mut.Lock()
		defer episodeStats.mut.Unlock()
		episodeStats.history = history.TotalRecords
		for _, m := range history.Data {
			if m.Score != "" {
				episodeStats.scores[m.Score]++
			}
			if m.Provider != "" {
				episodeStats.providers[m.Provider]++
			}
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting episodes subtitles", "error", err)
		return nil
	}

	et := time.Since(mseries)
	log.Debug("episode completed", "duration", et)

	return episodeStats
}

func (collector *bazarrCollector) collectMovieStats(ch chan<- prometheus.Metric, c *client.Client) *stats {
	log := slog.With("collector", "bazarr")
	mseries := time.Now()
	movieStats := new(stats)
	movieStats.languages = make(map[string]int)
	movieStats.scores = make(map[string]int)
	movieStats.providers = make(map[string]int)

	eg := errgroup.Group{}
	goRecoverable(&eg, func() error {
		history, err := client.Get[model.BazarrHistory](c, "movies/history")
		if err != nil {
			return fmt.Errorf("getting movies history: %w", err)
		}
		movieStats.mut.Lock()
		defer movieStats.mut.Unlock()
		movieStats.history = history.TotalRecords
		for _, m := range history.Data {
			if m.Score != "" {
				movieStats.scores[m.Score]++
			}
			if m.Provider != "" {
				movieStats.providers[m.Provider]++
			}
		}
		return nil
	})

	movies, err := client.Get[model.BazarrMovies](c, "movies")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting subtitles", "error", err)
		return nil
	}

	movieStats.mut.Lock()
	for _, m := range movies.Data {
		if !m.Monitored {
			movieStats.unmonitored++
		} else {
			movieStats.monitored++
		}

		if len(m.MissingSubtitles) > 0 {
			movieStats.missing += len(m.MissingSubtitles)
			if len(m.Subtitles) == 0 {
				movieStats.wanted++
			}
		}

		if len(m.Subtitles) > 0 {
			movieStats.downloaded += len(m.Subtitles)
			for _, subtitle := range m.Subtitles {
				if subtitle.Language != "" {
					movieStats.languages[subtitle.Language]++
				} else if subtitle.Code2 != "" {
					movieStats.languages[subtitle.Code2]++
				} else if subtitle.Code3 != "" {
					movieStats.languages[subtitle.Code3]++
				}

				if subtitle.Size != 0 {
					movieStats.fileSize += subtitle.Size
				}
			}
		}
	}

	movieStats.mut.Unlock()

	// Bazarr keeps separate histories for TV vs Movies, and therefore cannot
	// leverage the shared HistoryCollector; it was fetched concurrently above.
	if err := eg.Wait(); err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting movies history", "error", err)
		return nil
	}

	mt := time.Since(mseries)
	log.Debug("Movies completed", "duration", mt)

	return movieStats
}

func (collector *bazarrCollector) systemMetrics(ch chan<- prometheus.Metric, c *client.Client, badges *model.BazarrBadges) {
	log := slog.With("collector", "bazarr")

	health, err := client.Get[model.BazarrHealth](c, "system/health")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting health", "error", err)
		return
	}

	// Group metrics by issue & object
	// Bazarr uses it's own health message format, and therefore cannot leverage the shared HealthCollector
	if len(health.Data) > 0 {
		for _, s := range health.Data {
			ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(1),
				s.Issue, s.Object,
			)
		}
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemHealthMetric, prometheus.GaugeValue, float64(0), "", "")
	}

	// Bazarr uses it's own status format. and therefore cannot leverage the shared StatusCollector
	if _, err := client.Get[model.BazarrStatus](c, "system/status"); err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatusMetric, prometheus.GaugeValue, float64(0.0))
		log.Error("Error getting system status", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatusMetric, prometheus.GaugeValue, float64(1.0))
	}

	// Badges were fetched once in Collect; a fetch failure already raised the
	// error gauge there.
	if badges == nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(collector.throttledProvidersMetric, prometheus.GaugeValue, float64(badges.Providers))
	ch <- prometheus.MustNewConstMetric(collector.signalrConnectedMetric, prometheus.GaugeValue, boolToFloat(badges.SonarrSignalr == "LIVE"), "sonarr")
	ch <- prometheus.MustNewConstMetric(collector.signalrConnectedMetric, prometheus.GaugeValue, boolToFloat(badges.RadarrSignalr == "LIVE"), "radarr")
}

// boolToFloat converts a bool to a 0/1 gauge value.
func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
