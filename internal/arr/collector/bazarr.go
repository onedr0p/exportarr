package collector

import (
	"fmt"
	"github.com/shamelin/exportarr/internal/arr/model"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"go.uber.org/zap"
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

	systemHealthMetric *prometheus.Desc // Total number of health issues
	systemStatusMetric *prometheus.Desc // Total number of system statuses
	errorMetric        *prometheus.Desc // Error Description for use with InvalidMetric
}

func createIDBatches(ids []string, batchSize int) [][]string {
	if len(ids) == 0 {
		return [][]string{}
	}
	items := ids
	ret := [][]string{}
	for batchSize < len(items) {
		ret = append(ret, items[0:batchSize:batchSize])
		items = items[batchSize:]
	}
	return append(ret, items)
}

func NewBazarrCollector(c *config.ArrConfig) *bazarrCollector {
	subtitleText := "subtitles"
	episodeText := "episode"
	movieText := "movie"
	return &bazarrCollector{
		config: c,
		subtitlesHistoryMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_history_total", c.App, subtitleText),
			fmt.Sprintf("Total number of history %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesDownloadedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_downloaded_total", c.App, subtitleText),
			fmt.Sprintf("Total number of downloaded %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesMonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_monitored_total", c.App, subtitleText),
			fmt.Sprintf("Total number of monitored %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesUnmonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_unmonitored_total", c.App, subtitleText),
			fmt.Sprintf("Total number of unmonitored %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesWantedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_wanted_total", c.App, subtitleText),
			fmt.Sprintf("Total number of wanted %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesMissingMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_missing_total", c.App, subtitleText),
			fmt.Sprintf("Total number of missing %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesFileSizeMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_filesize_total", c.App, subtitleText),
			fmt.Sprintf("Total filesize of all %s", subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesHistoryMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_history_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of history %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesDownloadedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_downloaded_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of downloaded %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesMonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_monitored_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of monitored %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesUnmonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_unmonitored_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of unmonitored %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesWantedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_wanted_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of wanted %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesMissingMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_missing_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total number of missing %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		episodeSubtitlesFileSizeMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_filesize_total", c.App, episodeText, subtitleText),
			fmt.Sprintf("Total filesize of all %s %s", episodeText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesHistoryMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_history_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of history %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesDownloadedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_downloaded_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of downloaded %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesMonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_monitored_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of monitored %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesUnmonitoredMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_unmonitored_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of unmonitored %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesWantedMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_wanted_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of wanted %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesMissingMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_missing_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total number of missing %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieSubtitlesFileSizeMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_%s_filesize_total", c.App, movieText, subtitleText),
			fmt.Sprintf("Total filesize of all %s %s", movieText, subtitleText),
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitlesLanguageMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_language_total", c.App, subtitleText),
			fmt.Sprintf("Total number of downloaded %s by language", subtitleText),
			[]string{"language"},
			prometheus.Labels{"url": c.URL},
		),
		subtitlesScoreMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_score_total", c.App, subtitleText),
			fmt.Sprintf("Total number of downloaded %s by score", subtitleText),
			[]string{"score"},
			prometheus.Labels{"url": c.URL},
		),
		subtitlesProviderMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_%s_provider_total", c.App, subtitleText),
			fmt.Sprintf("Total number of downloaded %s by provider", subtitleText),
			[]string{"provider"},
			prometheus.Labels{"url": c.URL},
		),
		systemHealthMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_system_health_issues", c.App),
			"Total number of health issues by object and issue",
			[]string{"object", "issue"},
			prometheus.Labels{"url": c.URL},
		),
		systemStatusMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_system_status", c.App),
			"System Status",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			fmt.Sprintf("%s_collector_error", c.App),
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *bazarrCollector) Describe(ch chan<- *prometheus.Desc) {
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
}

func (collector *bazarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "bazarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	tseries := time.Now()

	collector.EpisodeMovieMetrics(ch, c)
	collector.SystemMetrics(ch, c)

	mt := time.Since(tseries)
	log.Debugw("All Completed", "duration", mt)
}

func (collector *bazarrCollector) EpisodeMovieMetrics(ch chan<- prometheus.Metric, c *client.Client) {

	episodeStats := newStats()
	if collector.config.EnableAdditionalMetrics {
		episodeStats = collector.CollectEpisodeStats(ch, c)
		if episodeStats == nil {
			return
		}
	}

	movieStats := collector.CollectMovieStats(ch, c)
	if movieStats == nil {
		return
	}

	if collector.config.EnableAdditionalMetrics {
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

	// just shove them into episode stats to avoid extra looping
	if len(movieStats.languages) > 0 {
		for languageName, count := range movieStats.languages {
			episodeStats.languages[languageName] += count
		}
	}

	if len(episodeStats.languages) > 0 {
		for languageName, count := range episodeStats.languages {
			ch <- prometheus.MustNewConstMetric(collector.subtitlesLanguageMetric, prometheus.GaugeValue, float64(count),
				languageName,
			)
		}
	}

	// just shove them into episode stats to avoid extra looping
	if len(movieStats.scores) > 0 {
		for score, count := range movieStats.scores {
			episodeStats.scores[score] += count
		}
	}

	if len(episodeStats.scores) > 0 {
		for score, count := range episodeStats.scores {
			ch <- prometheus.MustNewConstMetric(collector.subtitlesScoreMetric, prometheus.GaugeValue, float64(count),
				score,
			)
		}
	}

	// just shove them into episode stats to avoid extra looping
	if len(movieStats.providers) > 0 {
		for providerName, count := range movieStats.providers {
			episodeStats.providers[providerName] += count
		}
	}

	if len(episodeStats.providers) > 0 {
		for providerName, count := range episodeStats.providers {
			ch <- prometheus.MustNewConstMetric(collector.subtitlesProviderMetric, prometheus.GaugeValue, float64(count),
				providerName,
			)
		}
	}
}

func (collector *bazarrCollector) CollectEpisodeStats(ch chan<- prometheus.Metric, c *client.Client) *stats {
	log := zap.S().With("collector", "bazarr")
	episodeStats := newStats()

	mseries := time.Now()

	series := model.BazarrSeries{}
	if err := c.DoRequest("series", &series); err != nil {
		log.Errorw("Error getting series",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return nil
	}

	ids := []string{}
	for _, s := range series.Data {
		ids = append(ids, fmt.Sprintf("%d", s.Id))
	}
	batches := createIDBatches(ids, collector.config.Bazarr.SeriesBatchSize)

	eg := errgroup.Group{}
	sem := make(chan int, collector.config.Bazarr.SeriesBatchConcurrency) // limit concurrency via semaphore
	for _, batch := range batches {
		params := client.QueryParams{"seriesid[]": batch}
		sem <- 1 //
		eg.Go(func() error {
			defer func() { <-sem }()
			episodes := model.BazarrEpisodes{}
			if err := c.DoRequest("episodes", &episodes, params); err != nil {
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
	if err := eg.Wait(); err != nil {
		log.Errorw("Error getting episodes subtitles", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return nil
	}

	history := model.BazarrHistory{}
	if err := c.DoRequest("episodes/history", &history); err != nil {
		log.Errorw("Error getting episodes history",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return nil
	}
	episodeStats.history = history.TotalRecords

	for _, m := range history.Data {
		if m.Score != "" {
			episodeStats.scores[m.Score]++
		}
		if m.Provider != "" {
			episodeStats.providers[m.Provider]++
		}
	}

	et := time.Since(mseries)
	log.Debugw("episode completed", "duration", et)

	return episodeStats
}

func (collector *bazarrCollector) CollectMovieStats(ch chan<- prometheus.Metric, c *client.Client) *stats {
	log := zap.S().With("collector", "bazarr")
	mseries := time.Now()
	movieStats := new(stats)
	movieStats.languages = make(map[string]int)
	movieStats.scores = make(map[string]int)
	movieStats.providers = make(map[string]int)

	movies := model.BazarrMovies{}
	if err := c.DoRequest("movies", &movies); err != nil {
		log.Errorw("Error getting subtitles", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return nil
	}

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

	// Bazarr keeps separate histories for TV vs Movies, and therefore cannot leverage the shared HistoryCollector.
	history := model.BazarrHistory{}
	if err := c.DoRequest("movies/history", &history); err != nil {
		log.Errorw("Error getting movies history",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return nil
	}

	movieStats.history = history.TotalRecords

	for _, m := range history.Data {
		if m.Score != "" {
			movieStats.scores[m.Score]++
		}
		if m.Provider != "" {
			movieStats.providers[m.Provider]++
		}
	}

	mt := time.Since(mseries)
	log.Debugw("Movies completed", "duration", mt)

	return movieStats
}

func (collector *bazarrCollector) SystemMetrics(ch chan<- prometheus.Metric, c *client.Client) {
	log := zap.S().With("collector", "bazarr")

	health := model.BazarrHealth{}
	if err := c.DoRequest("system/health", &health); err != nil {
		log.Errorw("Error getting movies history",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
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
	systemStatus := model.BazarrStatus{}
	if err := c.DoRequest("system/status", &systemStatus); err != nil {
		ch <- prometheus.MustNewConstMetric(collector.systemStatusMetric, prometheus.GaugeValue, float64(0.0))
		log.Errorw("Error getting system status", "error", err)
	} else {
		ch <- prometheus.MustNewConstMetric(collector.systemStatusMetric, prometheus.GaugeValue, float64(1.0))
	}

}
