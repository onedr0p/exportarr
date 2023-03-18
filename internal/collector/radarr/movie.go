package collector

import (
	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type radarrCollector struct {
	config                 *config.Config   // App configuration
	movieMetric            *prometheus.Desc // Total number of movies
	movieDownloadedMetric  *prometheus.Desc // Total number of downloaded movies
	movieMonitoredMetric   *prometheus.Desc // Total number of monitored movies
	movieUnmonitoredMetric *prometheus.Desc // Total number of monitored movies
	movieWantedMetric      *prometheus.Desc // Total number of wanted movies
	movieMissingMetric     *prometheus.Desc // Total number of missing movies
	movieQualitiesMetric   *prometheus.Desc // Total number of movies by quality
	movieFileSizeMetric    *prometheus.Desc // Total fizesize of all movies in bytes
	errorMetric            *prometheus.Desc // Error Description for use with InvalidMetric
}

func NewRadarrCollector(c *config.Config) *radarrCollector {
	return &radarrCollector{
		config: c,
		movieMetric: prometheus.NewDesc(
			"radarr_movie_total",
			"Total number of movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieDownloadedMetric: prometheus.NewDesc(
			"radarr_movie_downloaded_total",
			"Total number of downloaded movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieMonitoredMetric: prometheus.NewDesc(
			"radarr_movie_monitored_total",
			"Total number of monitored movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieUnmonitoredMetric: prometheus.NewDesc(
			"radarr_movie_unmonitored_total",
			"Total number of unmonitored movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieWantedMetric: prometheus.NewDesc(
			"radarr_movie_wanted_total",
			"Total number of wanted movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieMissingMetric: prometheus.NewDesc(
			"radarr_movie_missing_total",
			"Total number of missing movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieFileSizeMetric: prometheus.NewDesc(
			"radarr_movie_filesize_total",
			"Total filesize of all movies",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		movieQualitiesMetric: prometheus.NewDesc(
			"radarr_movie_quality_total",
			"Total number of downloaded movies by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		errorMetric: prometheus.NewDesc(
			"radarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
	}
}

func (collector *radarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.movieMetric
	ch <- collector.movieDownloadedMetric
	ch <- collector.movieMonitoredMetric
	ch <- collector.movieUnmonitoredMetric
	ch <- collector.movieWantedMetric
	ch <- collector.movieMissingMetric
	ch <- collector.movieFileSizeMetric
	ch <- collector.movieQualitiesMetric
}

func (collector *radarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "radarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	var fileSize int64
	var (
		downloaded  = 0
		monitored   = 0
		unmonitored = 0
		missing     = 0
		wanted      = 0
		qualities   = map[string]int{}
	)
	movies := model.Movie{}
	// https://radarr.video/docs/api/#/Movie/get_api_v3_movie
	if err := c.DoRequest("movie", &movies); err != nil {
		log.Errorw("Error getting movies", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, s := range movies {
		if s.HasFile {
			downloaded++
		}
		if !s.Monitored {
			unmonitored++
		} else {
			monitored++
			if !s.HasFile && s.Available {
				missing++
			} else if !s.HasFile {
				wanted++
			}
		}

		if s.MovieFile.Quality.Quality.Name != "" {
			qualities[s.MovieFile.Quality.Quality.Name]++
		}
		if s.MovieFile.Size != 0 {
			fileSize += s.MovieFile.Size
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.movieMetric, prometheus.GaugeValue, float64(len(movies)))
	ch <- prometheus.MustNewConstMetric(collector.movieDownloadedMetric, prometheus.GaugeValue, float64(downloaded))
	ch <- prometheus.MustNewConstMetric(collector.movieMonitoredMetric, prometheus.GaugeValue, float64(monitored))
	ch <- prometheus.MustNewConstMetric(collector.movieUnmonitoredMetric, prometheus.GaugeValue, float64(unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.movieWantedMetric, prometheus.GaugeValue, float64(wanted))
	ch <- prometheus.MustNewConstMetric(collector.movieMissingMetric, prometheus.GaugeValue, float64(missing))
	ch <- prometheus.MustNewConstMetric(collector.movieFileSizeMetric, prometheus.GaugeValue, float64(fileSize))

	if len(qualities) > 0 {
		for qualityName, count := range qualities {
			ch <- prometheus.MustNewConstMetric(collector.movieQualitiesMetric, prometheus.GaugeValue, float64(count),
				qualityName,
			)
		}
	}
}
