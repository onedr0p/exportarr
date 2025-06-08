package collector

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type sonarrCollector struct {
	config                   *config.ArrConfig // App configuration
	seriesMetric             *prometheus.Desc  // Total number of series
	seriesDownloadedMetric   *prometheus.Desc  // Total number of downloaded series
	seriesMonitoredMetric    *prometheus.Desc  // Total number of monitored series
	seriesUnmonitoredMetric  *prometheus.Desc  // Total number of unmonitored series
	seriesFileSizeMetric     *prometheus.Desc  // Total fizesize of all series in bytes
	seasonMetric             *prometheus.Desc  // Total number of seasons
	seasonDownloadedMetric   *prometheus.Desc  // Total number of downloaded seasons
	seasonMonitoredMetric    *prometheus.Desc  // Total number of monitored seasons
	seasonUnmonitoredMetric  *prometheus.Desc  // Total number of unmonitored seasons
	episodeMetric            *prometheus.Desc  // Total number of episodes
	episodeMonitoredMetric   *prometheus.Desc  // Total number of monitored episodes
	episodeUnmonitoredMetric *prometheus.Desc  // Total number of unmonitored episodes
	episodeDownloadedMetric  *prometheus.Desc  // Total number of downloaded episodes
	episodeMissingMetric     *prometheus.Desc  // Total number of missing episodes
	episodeQualitiesMetric   *prometheus.Desc  // Total number of episodes by quality
	errorMetric              *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewSonarrCollector(conf *config.ArrConfig) *sonarrCollector {
	return &sonarrCollector{
		config: conf,
		seriesMetric: prometheus.NewDesc(
			"sonarr_series_total",
			"Total number of series",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seriesDownloadedMetric: prometheus.NewDesc(
			"sonarr_series_downloaded_total",
			"Total number of downloaded series",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seriesMonitoredMetric: prometheus.NewDesc(
			"sonarr_series_monitored_total",
			"Total number of monitored series",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seriesUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_series_unmonitored_total",
			"Total number of unmonitored series",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seriesFileSizeMetric: prometheus.NewDesc(
			"sonarr_series_filesize_bytes",
			"Total fizesize of all series in bytes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seasonMetric: prometheus.NewDesc(
			"sonarr_season_total",
			"Total number of seasons",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seasonDownloadedMetric: prometheus.NewDesc(
			"sonarr_season_downloaded_total",
			"Total number of downloaded seasons",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seasonMonitoredMetric: prometheus.NewDesc(
			"sonarr_season_monitored_total",
			"Total number of monitored seasons",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		seasonUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_season_unmonitored_total",
			"Total number of unmonitored seasons",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeMetric: prometheus.NewDesc(
			"sonarr_episode_total",
			"Total number of episodes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeMonitoredMetric: prometheus.NewDesc(
			"sonarr_episode_monitored_total",
			"Total number of monitored episodes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_episode_unmonitored_total",
			"Total number of unmonitored episodes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeDownloadedMetric: prometheus.NewDesc(
			"sonarr_episode_downloaded_total",
			"Total number of downloaded episodes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeMissingMetric: prometheus.NewDesc(
			"sonarr_episode_missing_total",
			"Total number of missing episodes",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
		episodeQualitiesMetric: prometheus.NewDesc(
			"sonarr_episode_quality_total",
			"Total number of downloaded episodes by quality",
			[]string{"quality", "weight"},
			prometheus.Labels{"url": conf.URL},
		),
		errorMetric: prometheus.NewDesc(
			"sonarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": conf.URL},
		),
	}
}

func (collector *sonarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.seriesMetric
	ch <- collector.seriesDownloadedMetric
	ch <- collector.seriesMonitoredMetric
	ch <- collector.seriesUnmonitoredMetric
	ch <- collector.seriesFileSizeMetric
	ch <- collector.seasonMetric
	ch <- collector.seasonDownloadedMetric
	ch <- collector.seasonMonitoredMetric
	ch <- collector.seasonUnmonitoredMetric
	ch <- collector.episodeMetric
	ch <- collector.episodeMonitoredMetric
	ch <- collector.episodeUnmonitoredMetric
	ch <- collector.episodeDownloadedMetric
	ch <- collector.episodeMissingMetric
	ch <- collector.episodeQualitiesMetric
}

func (collector *sonarrCollector) Collect(ch chan<- prometheus.Metric) {
	total := time.Now()
	log := zap.S().With("collector", "sonarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
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

	cseries := []time.Duration{}
	series := model.Series{}
	if err := c.DoRequest("series", &series); err != nil {
		log.Errorw("Error getting series",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	for _, s := range series {
		tseries := time.Now()

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

		if collector.config.EnableAdditionalMetrics {
			textra := time.Now()
			episodeFile := model.EpisodeFile{}

			params := client.QueryParams{}
			params.Add("seriesId", fmt.Sprintf("%d", s.Id))

			if err := c.DoRequest("episodefile", &episodeFile, params); err != nil {
				log.Errorw("Error getting episodefile",
					"error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			for _, e := range episodeFile {
				if e.Quality.Quality.Name != "" {
					episodesQualities[e.Quality.Quality.Name]++
				}
			}

			episode := model.Episode{}
			if err := c.DoRequest("episode", &episode, params); err != nil {
				log.Errorw("Error getting episode",
					"error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			for _, e := range episode {
				if e.Monitored {
					episodesMonitored++
				} else {
					episodesUnmonitored++
				}
			}

			qualities := model.Qualities{}
			if err := c.DoRequest("qualitydefinition", &qualities); err != nil {
				log.Errorw("Error getting qualities",
					"error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			for _, q := range qualities {
				if q.Quality.Name != "" {
					qualityWeights[q.Quality.Name] = strconv.Itoa(q.Weight)
				}
			}

			log.Debugw("Extra options completed",
				"duration", time.Since(textra))
		}
		e := time.Since(tseries)
		cseries = append(cseries, e)
		log.Debugw("series completed",
			"series_id", s.Id,
			"duration", e)
	}

	episodesMissing := model.Missing{}

	params := client.QueryParams{}
	params.Add("sortKey", "airDateUtc")

	if err := c.DoRequest("wanted/missing", &episodesMissing, params); err != nil {
		log.Errorw("Error getting missing",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(collector.seriesMetric, prometheus.GaugeValue, float64(len(series)))
	ch <- prometheus.MustNewConstMetric(collector.seriesDownloadedMetric, prometheus.GaugeValue, float64(seriesDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.seriesMonitoredMetric, prometheus.GaugeValue, float64(seriesMonitored))
	ch <- prometheus.MustNewConstMetric(collector.seriesUnmonitoredMetric, prometheus.GaugeValue, float64(seriesUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.seriesFileSizeMetric, prometheus.GaugeValue, float64(seriesFileSize))
	ch <- prometheus.MustNewConstMetric(collector.seasonMetric, prometheus.GaugeValue, float64(seasons))
	ch <- prometheus.MustNewConstMetric(collector.seasonDownloadedMetric, prometheus.GaugeValue, float64(seasonsDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.seasonMonitoredMetric, prometheus.GaugeValue, float64(seasonsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.seasonUnmonitoredMetric, prometheus.GaugeValue, float64(seasonsUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.episodeMetric, prometheus.GaugeValue, float64(episodes))
	ch <- prometheus.MustNewConstMetric(collector.episodeDownloadedMetric, prometheus.GaugeValue, float64(episodesDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.episodeMissingMetric, prometheus.GaugeValue, float64(episodesMissing.TotalRecords))

	if collector.config.EnableAdditionalMetrics {
		ch <- prometheus.MustNewConstMetric(collector.episodeMonitoredMetric, prometheus.GaugeValue, float64(episodesMonitored))
		ch <- prometheus.MustNewConstMetric(collector.episodeUnmonitoredMetric, prometheus.GaugeValue, float64(episodesUnmonitored))

		if len(episodesQualities) > 0 {
			for qualityName, count := range episodesQualities {
				ch <- prometheus.MustNewConstMetric(collector.episodeQualitiesMetric, prometheus.GaugeValue, float64(count),
					qualityName, qualityWeights[qualityName],
				)
			}
		}
	}
	log.Debugw("Sonarr cycle completed",
		"duration", time.Since(total),
		"series_durations", cseries,
	)

}
