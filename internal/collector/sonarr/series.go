package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type sonarrCollector struct {
	config                  *cli.Context     // App configuration
	seriesMetric            *prometheus.Desc // Total number of series
	seriesMonitoredMetric   *prometheus.Desc // Total number of monitored series
	seriesFileSizeMetric    *prometheus.Desc // Total fizesize of all series in bytes
	seasonMetric            *prometheus.Desc // Total number of seasons
	seasonMonitoredMetric   *prometheus.Desc // Total number of monitored seasons
	episodeMetric           *prometheus.Desc // Total number of episodes
	episodeMonitoredMetric  *prometheus.Desc // Total number of monitored episodes
	episodeDownloadedMetric *prometheus.Desc // Total number of downloaded episodes
	episodeMissingMetric    *prometheus.Desc // Total number of missing episodes
	episodeQualitiesMetric  *prometheus.Desc // Total number of episodes by quality
}

func NewSonarrCollector(c *cli.Context) *sonarrCollector {
	return &sonarrCollector{
		config: c,
		seriesMetric: prometheus.NewDesc(
			"sonarr_series_total",
			"Total number of series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesMonitoredMetric: prometheus.NewDesc(
			"sonarr_series_monitored_total",
			"Total number of monitored series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seriesFileSizeMetric: prometheus.NewDesc(
			"sonarr_series_filesize_bytes",
			"Total number of monitored series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonMetric: prometheus.NewDesc(
			"sonarr_season_total",
			"Total number of series",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		seasonMonitoredMetric: prometheus.NewDesc(
			"sonarr_season_monitored_total",
			"Total number of series",
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
	ch <- collector.seriesMonitoredMetric
	ch <- collector.seriesFileSizeMetric
	ch <- collector.seasonMetric
	ch <- collector.seasonMonitoredMetric
	ch <- collector.episodeMetric
	ch <- collector.episodeMonitoredMetric
	ch <- collector.episodeDownloadedMetric
	ch <- collector.episodeMissingMetric
	ch <- collector.episodeQualitiesMetric
}

func (collector *sonarrCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config)
	var seriesFileSize int64
	var (
		seriesMonitored    = 0
		seasons            = 0
		seasonsMonitored   = 0
		episodes           = 0
		episodesDownloaded = 0
		episodesQualities  = map[string]int{}
	)
	series := model.Series{}
	if err := c.DoRequest("series", &series); err != nil {
		log.Fatal(err)
	}
	for _, s := range series {
		if s.Monitored {
			seriesMonitored++
		}
		seasons += s.Statistics.SeasonCount
		episodes += s.Statistics.TotalEpisodeCount
		episodesDownloaded += s.Statistics.EpisodeFileCount
		seriesFileSize += s.Statistics.SizeOnDisk
		for _, e := range s.Seasons {
			if e.Monitored {
				seasonsMonitored++
			}
		}
		if !collector.config.Bool("disable-episode-quality-metrics") {
			episodeFile := model.EpisodeFile{}
			if err := c.DoRequest(fmt.Sprintf("%s?seriesId=%d", "episodefile", s.Id), &episodeFile); err != nil {
				log.Fatal(err)
			}
			for _, e := range episodeFile {
				if e.Quality.Quality.Name != "" {
					episodesQualities[e.Quality.Quality.Name]++
				}
			}
		}
	}

	missing := model.Missing{}
	if err := c.DoRequest("wanted/missing?sortKey=airDateUtc", &missing); err != nil {
		log.Fatal(err)
	}

	ch <- prometheus.MustNewConstMetric(collector.seriesMetric, prometheus.GaugeValue, float64(len(series)))
	ch <- prometheus.MustNewConstMetric(collector.seriesMonitoredMetric, prometheus.GaugeValue, float64(seriesMonitored))
	ch <- prometheus.MustNewConstMetric(collector.seriesFileSizeMetric, prometheus.GaugeValue, float64(seriesFileSize))
	ch <- prometheus.MustNewConstMetric(collector.seasonMetric, prometheus.GaugeValue, float64(seasons))
	ch <- prometheus.MustNewConstMetric(collector.seasonMonitoredMetric, prometheus.GaugeValue, float64(seasonsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.episodeMetric, prometheus.GaugeValue, float64(episodes))
	ch <- prometheus.MustNewConstMetric(collector.episodeDownloadedMetric, prometheus.GaugeValue, float64(episodesDownloaded))

	if !collector.config.Bool("disable-episode-quality-metrics") {
		if len(episodesQualities) > 0 {
			for qualityName, count := range episodesQualities {
				ch <- prometheus.MustNewConstMetric(collector.episodeQualitiesMetric, prometheus.GaugeValue, float64(count),
					qualityName,
				)
			}
		}
	}
}
