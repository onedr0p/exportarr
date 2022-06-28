package collector

import (
	"fmt"
	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"time"
)

type sonarrCollector struct {
	config                   *cli.Context     // App configuration
	configFile               *model.Config    // *arr configuration from config.xml
	seriesMetric             *prometheus.Desc // Total number of series
	seriesDownloadedMetric   *prometheus.Desc // Total number of downloaded series
	seriesMonitoredMetric    *prometheus.Desc // Total number of monitored series
	seriesUnmonitoredMetric  *prometheus.Desc // Total number of unmonitored series
	seriesFileSizeMetric     *prometheus.Desc // Total fizesize of all series in bytes
	seasonMetric             *prometheus.Desc // Total number of seasons
	seasonDownloadedMetric   *prometheus.Desc // Total number of downloaded seasons
	seasonMonitoredMetric    *prometheus.Desc // Total number of monitored seasons
	seasonUnmonitoredMetric  *prometheus.Desc // Total number of monitored seasons
	episodeMetric            *prometheus.Desc // Total number of episodes
	episodeMonitoredMetric   *prometheus.Desc // Total number of monitored episodes
	episodeUnmonitoredMetric *prometheus.Desc // Total number of unmonitored episodes
	episodeDownloadedMetric  *prometheus.Desc // Total number of downloaded episodes
	episodeMissingMetric     *prometheus.Desc // Total number of missing episodes
	episodeQualitiesMetric   *prometheus.Desc // Total number of episodes by quality
}

func NewSonarrCollector(c *cli.Context, cf *model.Config) *sonarrCollector {
	return &sonarrCollector{
		config:     c,
		configFile: cf,
		seriesMetric: prometheus.NewDesc(
			"sonarr_series_total",
			"Total number of series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesDownloadedMetric: prometheus.NewDesc(
			"sonarr_series_downloaded_total",
			"Total number of downloaded series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesMonitoredMetric: prometheus.NewDesc(
			"sonarr_series_monitored_total",
			"Total number of monitored series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_series_unmonitored_total",
			"Total number of unmonitored series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesFileSizeMetric: prometheus.NewDesc(
			"sonarr_series_filesize_bytes",
			"Total fizesize of all series in bytes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonMetric: prometheus.NewDesc(
			"sonarr_season_total",
			"Total number of seasons",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonDownloadedMetric: prometheus.NewDesc(
			"sonarr_season_downloaded_total",
			"Total number of downloaded seasons",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonMonitoredMetric: prometheus.NewDesc(
			"sonarr_season_monitored_total",
			"Total number of monitored seasons",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_season_unmonitored_total",
			"Total number of unmonitored seasons",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeMetric: prometheus.NewDesc(
			"sonarr_episode_total",
			"Total number of episodes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeMonitoredMetric: prometheus.NewDesc(
			"sonarr_episode_monitored_total",
			"Total number of monitored episodes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeUnmonitoredMetric: prometheus.NewDesc(
			"sonarr_episode_unmonitored_total",
      "Total number of unmonitored episodes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeDownloadedMetric: prometheus.NewDesc(
			"sonarr_episode_downloaded_total",
			"Total number of downloaded episodes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeMissingMetric: prometheus.NewDesc(
			"sonarr_episode_missing_total",
			"Total number of missing episodes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		episodeQualitiesMetric: prometheus.NewDesc(
			"sonarr_episode_quality_total",
			"Total number of downloaded episodes by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.String("url")},
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
	c := client.NewClient(collector.config, collector.configFile)
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
		episodeMonitored    = 0
		episodesUnmonitored = 0
		episodesQualities   = map[string]int{}
	)

	cseries := []time.Duration{}
	series := model.Series{}
	if err := c.DoRequest("series", &series); err != nil {
		log.Fatal(err)
	}

	for _, s := range series {
		tseries := time.Now()

		if !s.Monitored {
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
			if !e.Monitored {
				seasonsUnmonitored++
			} else {
				seasonsMonitored++
			}

			if e.Statistics.PercentOfEpisodes == 100 {
				seasonsDownloaded++
			}
		}

		if collector.config.Bool("enable-additional-metrics") {
			textra := time.Now()
			episodeFile := model.EpisodeFile{}
			if err := c.DoRequest(fmt.Sprintf("%s?seriesId=%d", "episodefile", s.Id), &episodeFile); err != nil {
				log.Fatal(err)
			}
			for _, e := range episodeFile {
				if e.Quality.Quality.Name != "" {
					episodesQualities[e.Quality.Quality.Name]++
				}
			}

			episode := model.Episode{}
			if err := c.DoRequest(fmt.Sprintf("%s?seriesId=%d", "episode", s.Id), &episode); err != nil {
				log.Fatal(err)
			}
			for _, e := range episode {
				if !e.Monitored {
					episodesUnmonitored++
				} else {
					episodesMonitored++
				}
			}
			log.Debug("TIME :: Extra options took %s", time.Since(textra))
		}
		e := time.Since(tseries)
		cseries = append(cseries, e)
		log.Debug("TIME :: series %s took %s", s.Id, e)
	}

	episodesMissing := model.Missing{}
	if err := c.DoRequest("wanted/missing?sortKey=airDateUtc", &episodesMissing); err != nil {
		log.Fatal(err)
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
	ch <- prometheus.MustNewConstMetric(collector.episodeMonitoredMetric, prometheus.GaugeValue, float64(episodesMonitored))
	ch <- prometheus.MustNewConstMetric(collector.episodeUnmonitoredMetric, prometheus.GaugeValue, float64(episodesUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.episodeMissingMetric, prometheus.GaugeValue, float64(episodesMissing.TotalRecords))

	if collector.config.Bool("enable-additional-metrics") {
		ch <- prometheus.MustNewConstMetric(collector.episodeMonitoredMetric, prometheus.GaugeValue, float64(episodesMonitored))
		ch <- prometheus.MustNewConstMetric(collector.episodeUnmonitoredMetric, prometheus.GaugeValue, float64(episodesUnmonitored))

		if len(episodesQualities) > 0 {
			for qualityName, count := range episodesQualities {
				ch <- prometheus.MustNewConstMetric(collector.episodeQualitiesMetric, prometheus.GaugeValue, float64(count),
					qualityName,
				)
			}
		}
	}
	log.Debug("TIME :: total took %s with series timings as %s",
		time.Since(total),
		cseries,
	)

}
