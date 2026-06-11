package collector

import (
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type sonarrCollector struct {
	collectMu                sync.Mutex // Guards against overlapping collections (#380)
	client                   *client.Client
	config                   *config.ArrConfig // App configuration
	seriesMetric             *prometheus.Desc  // Total number of series
	seriesDownloadedMetric   *prometheus.Desc  // Total number of downloaded series
	seriesMonitoredMetric    *prometheus.Desc  // Total number of monitored series
	seriesUnmonitoredMetric  *prometheus.Desc  // Total number of unmonitored series
	seriesFileSizeMetric     *prometheus.Desc  // Total fizesize of all series in bytes
	seriesTagsMetric         *prometheus.Desc  // Total number of series by tag
	seasonMetric             *prometheus.Desc  // Total number of seasons
	seasonDownloadedMetric   *prometheus.Desc  // Total number of downloaded seasons
	seasonMonitoredMetric    *prometheus.Desc  // Total number of monitored seasons
	seasonUnmonitoredMetric  *prometheus.Desc  // Total number of unmonitored seasons
	episodeMetric            *prometheus.Desc  // Total number of episodes
	episodeMonitoredMetric   *prometheus.Desc  // Total number of monitored episodes
	episodeUnmonitoredMetric *prometheus.Desc  // Total number of unmonitored episodes
	episodeDownloadedMetric  *prometheus.Desc  // Total number of downloaded episodes
	episodeMissingMetric     *prometheus.Desc  // Total number of missing episodes
	episodeCutoffUnmetMetric *prometheus.Desc  // Total number of episodes with cutoff unmet
	episodeQualitiesMetric   *prometheus.Desc  // Total number of episodes by quality
	errorMetric              *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewSonarrCollector builds a collector for sonarr library statistics.
func NewSonarrCollector(httpClient *client.Client, conf *config.ArrConfig) prometheus.Collector {
	return &sonarrCollector{
		client:                   httpClient,
		config:                   conf,
		seriesMetric:             newDesc("sonarr", "series_total", "Total number of series", nil, conf.URL),
		seriesDownloadedMetric:   newDesc("sonarr", "series_downloaded_total", "Total number of downloaded series", nil, conf.URL),
		seriesMonitoredMetric:    newDesc("sonarr", "series_monitored_total", "Total number of monitored series", nil, conf.URL),
		seriesUnmonitoredMetric:  newDesc("sonarr", "series_unmonitored_total", "Total number of unmonitored series", nil, conf.URL),
		seriesFileSizeMetric:     newDesc("sonarr", "series_filesize_bytes", "Total fizesize of all series in bytes", nil, conf.URL),
		seriesTagsMetric:         newDesc("sonarr", "series_tag_total", "Total number of downloaded series by tag", []string{"tag"}, conf.URL),
		seasonMetric:             newDesc("sonarr", "season_total", "Total number of seasons", nil, conf.URL),
		seasonDownloadedMetric:   newDesc("sonarr", "season_downloaded_total", "Total number of downloaded seasons", nil, conf.URL),
		seasonMonitoredMetric:    newDesc("sonarr", "season_monitored_total", "Total number of monitored seasons", nil, conf.URL),
		seasonUnmonitoredMetric:  newDesc("sonarr", "season_unmonitored_total", "Total number of unmonitored seasons", nil, conf.URL),
		episodeMetric:            newDesc("sonarr", "episode_total", "Total number of episodes", nil, conf.URL),
		episodeMonitoredMetric:   newDesc("sonarr", "episode_monitored_total", "Total number of monitored episodes", nil, conf.URL),
		episodeUnmonitoredMetric: newDesc("sonarr", "episode_unmonitored_total", "Total number of unmonitored episodes", nil, conf.URL),
		episodeDownloadedMetric:  newDesc("sonarr", "episode_downloaded_total", "Total number of downloaded episodes", nil, conf.URL),
		episodeMissingMetric:     newDesc("sonarr", "episode_missing_total", "Total number of missing episodes", nil, conf.URL),
		episodeCutoffUnmetMetric: newDesc("sonarr", "episode_cutoff_unmet_total", "Total number of episodes with cutoff unmet", nil, conf.URL),
		episodeQualitiesMetric:   newDesc("sonarr", "episode_quality_total", "Total number of downloaded episodes by quality", []string{"quality", "weight"}, conf.URL),
		errorMetric:              newDesc("sonarr", "collector_error", "Error while collecting metrics", nil, conf.URL),
	}
}

func (collector *sonarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.seriesMetric
	ch <- collector.seriesDownloadedMetric
	ch <- collector.seriesMonitoredMetric
	ch <- collector.seriesUnmonitoredMetric
	ch <- collector.seriesFileSizeMetric
	ch <- collector.seriesTagsMetric
	ch <- collector.seasonMetric
	ch <- collector.seasonDownloadedMetric
	ch <- collector.seasonMonitoredMetric
	ch <- collector.seasonUnmonitoredMetric
	ch <- collector.episodeMetric
	ch <- collector.episodeMonitoredMetric
	ch <- collector.episodeUnmonitoredMetric
	ch <- collector.episodeDownloadedMetric
	ch <- collector.episodeMissingMetric
	ch <- collector.episodeCutoffUnmetMetric
	ch <- collector.episodeQualitiesMetric
}

func (collector *sonarrCollector) Collect(ch chan<- prometheus.Metric) {
	total := time.Now()
	log := slog.With("collector", "sonarr")
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
	c, err := client.NewClient(collector.config)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error creating client", "error", err)
		return
	}
	var seriesFileSize int64
	var (
		seriesDownloaded    = 0
		seriesMonitored     = 0
		seriesUnmonitored   = 0
		seasons             = 0
		seasonsDownloaded   = 0
		seasonsMonitored    = 0
		seasonsUnmonitored  = 0
		episodes            = 0
		episodesDownloaded  = 0
		episodesMonitored   = 0
		episodesUnmonitored = 0
		episodesQualities   = map[string]int{}
		qualityWeights      = map[string]string{}
	)

	series, err := client.Get[model.Series](c, "series")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting series", "error", err)
		return
	}

	collectQuality := !collector.config.DisableQualityMetrics
	collectEpisodes := !collector.config.DisableEpisodeMetrics

	// Quality definitions are repository-global: fetch once, not per series.
	if collectQuality {
		qualities, err := client.Get[model.Qualities](c, "qualitydefinition")
		if err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting qualities", "error", err)
			return
		}
		for _, q := range qualities {
			if q.Quality.Name != "" {
				qualityWeights[q.Quality.Name] = strconv.Itoa(q.Weight)
			}
		}
	}

	for _, s := range series {
		if s.Monitored {
			seriesMonitored++
		} else {
			seriesUnmonitored++
		}

		if s.Statistics.PercentOfEpisodes == 100 {
			seriesDownloaded++
		}

		seasons += s.Statistics.SeasonCount
		episodes += s.Statistics.TotalEpisodeCount
		episodesDownloaded += s.Statistics.EpisodeFileCount
		seriesFileSize += s.Statistics.SizeOnDisk

		for _, e := range s.Seasons {
			if e.Monitored {
				seasonsMonitored++
			} else {
				seasonsUnmonitored++
			}

			if e.Statistics.PercentOfEpisodes == 100 {
				seasonsDownloaded++
			}
		}
	}

	// The per-series episode lookups dominate scrape time on large libraries:
	// fan them out with bounded concurrency instead of ~2×N serial requests.
	if collectQuality || collectEpisodes {
		var mu sync.Mutex
		eg := errgroup.Group{}
		eg.SetLimit(maxConcurrentSeriesFetches)
		for _, s := range series {
			goRecoverable(&eg, func() error {
				params := client.QueryParams{}
				params.Add("seriesId", strconv.Itoa(s.ID))

				if collectQuality {
					episodeFile, err := client.Get[model.EpisodeFile](c, "episodefile", params)
					if err != nil {
						return fmt.Errorf("getting episodefile for series %d: %w", s.ID, err)
					}
					mu.Lock()
					for _, e := range episodeFile {
						if e.Quality.Quality.Name != "" {
							episodesQualities[e.Quality.Quality.Name]++
						}
					}
					mu.Unlock()
				}
				if collectEpisodes {
					episode, err := client.Get[model.Episode](c, "episode", params)
					if err != nil {
						return fmt.Errorf("getting episode for series %d: %w", s.ID, err)
					}
					mu.Lock()
					for _, e := range episode {
						if e.Monitored {
							episodesMonitored++
						} else {
							episodesUnmonitored++
						}
					}
					mu.Unlock()
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting per-series episode metrics", "error", err)
			return
		}
	}

	// Only totalRecords is read from the wanted endpoints: request a single
	// record and skip server-side sorting entirely — *arr sorts slowly, and
	// ordering is Prometheus/Grafana's job anyway. These totals force full
	// table counts, so they are skippable on huge instances.
	var episodesMissing, episodesCutoffUnmet int
	if !collector.config.DisableWantedMetrics {
		params := client.QueryParams{}
		params.Add("pageSize", "1")

		missing, err := client.Get[model.Missing](c, "wanted/missing", params)
		if err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting missing", "error", err)
			return
		}
		episodesMissing = missing.TotalRecords

		// Cutoff unmet endpoint uses the same params as missing
		cutoffUnmet, err := client.Get[model.CutoffUnmet](c, "wanted/cutoff", params)
		if err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting cutoff unmet", "error", err)
			return
		}
		episodesCutoffUnmet = cutoffUnmet.TotalRecords
	}

	// Get tag details for series
	tagObjects, err := client.Get[model.TagSeries](c, "tag/detail")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting tags", "error", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(collector.seriesMetric, prometheus.GaugeValue, float64(len(series)))
	ch <- prometheus.MustNewConstMetric(collector.seriesDownloadedMetric, prometheus.GaugeValue, float64(seriesDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.seriesMonitoredMetric, prometheus.GaugeValue, float64(seriesMonitored))
	ch <- prometheus.MustNewConstMetric(collector.seriesUnmonitoredMetric, prometheus.GaugeValue, float64(seriesUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.seriesFileSizeMetric, prometheus.GaugeValue, float64(seriesFileSize))
	for _, tag := range tagObjects {
		ch <- prometheus.MustNewConstMetric(collector.seriesTagsMetric, prometheus.GaugeValue, float64(len(tag.SeriesIDs)),
			tag.Label,
		)
	}
	ch <- prometheus.MustNewConstMetric(collector.seasonMetric, prometheus.GaugeValue, float64(seasons))
	ch <- prometheus.MustNewConstMetric(collector.seasonDownloadedMetric, prometheus.GaugeValue, float64(seasonsDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.seasonMonitoredMetric, prometheus.GaugeValue, float64(seasonsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.seasonUnmonitoredMetric, prometheus.GaugeValue, float64(seasonsUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.episodeMetric, prometheus.GaugeValue, float64(episodes))
	ch <- prometheus.MustNewConstMetric(collector.episodeDownloadedMetric, prometheus.GaugeValue, float64(episodesDownloaded))
	if !collector.config.DisableWantedMetrics {
		ch <- prometheus.MustNewConstMetric(collector.episodeMissingMetric, prometheus.GaugeValue, float64(episodesMissing))
		ch <- prometheus.MustNewConstMetric(collector.episodeCutoffUnmetMetric, prometheus.GaugeValue, float64(episodesCutoffUnmet))
	}

	if collectEpisodes {
		ch <- prometheus.MustNewConstMetric(collector.episodeMonitoredMetric, prometheus.GaugeValue, float64(episodesMonitored))
		ch <- prometheus.MustNewConstMetric(collector.episodeUnmonitoredMetric, prometheus.GaugeValue, float64(episodesUnmonitored))
	}
	if collectQuality && len(episodesQualities) > 0 {
		for qualityName, count := range episodesQualities {
			ch <- prometheus.MustNewConstMetric(collector.episodeQualitiesMetric, prometheus.GaugeValue, float64(count),
				qualityName, qualityWeights[qualityName],
			)
		}
	}
	log.Debug("Sonarr cycle completed", "duration", time.Since(total))

}
