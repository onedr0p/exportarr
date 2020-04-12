package collector

import (
	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type movieCollector struct {
	config            *cli.Context
	movieMetric       *prometheus.Desc
	downloadedMetric  *prometheus.Desc
	monitoredMetric   *prometheus.Desc
	unmonitoredMetric *prometheus.Desc
	wantedMetric      *prometheus.Desc
	missingMetric     *prometheus.Desc
	filesizeMetric    *prometheus.Desc
	qualitiesMetric   *prometheus.Desc
}

func NewMovieCollector(c *cli.Context) *movieCollector {
	return &movieCollector{
		config: c,
		movieMetric: prometheus.NewDesc(
			"radarr_movie_total",
			"Total number of movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		downloadedMetric: prometheus.NewDesc(
			"radarr_movie_download_total",
			"Total number of downloaded movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		monitoredMetric: prometheus.NewDesc(
			"radarr_movie_monitored_total",
			"Total number of monitored movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		unmonitoredMetric: prometheus.NewDesc(
			"radarr_movie_unmonitored_total",
			"Total number of unmonitored movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		wantedMetric: prometheus.NewDesc(
			"radarr_movie_wanted_total",
			"Total number of wanted movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		missingMetric: prometheus.NewDesc(
			"radarr_movie_missing_total",
			"Total number of missing movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		filesizeMetric: prometheus.NewDesc(
			"radarr_movie_filesize_total",
			"Total filesize of all movies",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		qualitiesMetric: prometheus.NewDesc(
			"radarr_movie_quality_total",
			"Total number of downloaded movies by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.String("url")},
		),
	}
}

func (collector *movieCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.movieMetric
	ch <- collector.downloadedMetric
	ch <- collector.monitoredMetric
	ch <- collector.unmonitoredMetric
	ch <- collector.wantedMetric
	ch <- collector.missingMetric
	ch <- collector.filesizeMetric
	ch <- collector.qualitiesMetric
}

func (collector *movieCollector) Collect(ch chan<- prometheus.Metric) {
	c := client.NewClient(collector.config)
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
	if err := c.DoRequest("movie", &movies); err != nil {
		log.Fatal(err)
	}
	for _, s := range movies {
		if s.HasFile {
			downloaded++
		}
		if s.Monitored {
			monitored++
			if !s.HasFile && s.Status == "released" {
				missing++
			} else if !s.HasFile {
				wanted++
			}
		} else {
			unmonitored++
		}
		if s.MovieFile.Quality.Quality.Name != "" {
			qualities[s.MovieFile.Quality.Quality.Name]++
		}
		if s.MovieFile.Size != 0 {
			fileSize += s.MovieFile.Size
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.movieMetric, prometheus.GaugeValue, float64(len(movies)))
	ch <- prometheus.MustNewConstMetric(collector.downloadedMetric, prometheus.GaugeValue, float64(downloaded))
	ch <- prometheus.MustNewConstMetric(collector.monitoredMetric, prometheus.GaugeValue, float64(monitored))
	ch <- prometheus.MustNewConstMetric(collector.unmonitoredMetric, prometheus.GaugeValue, float64(unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.wantedMetric, prometheus.GaugeValue, float64(wanted))
	ch <- prometheus.MustNewConstMetric(collector.missingMetric, prometheus.GaugeValue, float64(missing))
	ch <- prometheus.MustNewConstMetric(collector.filesizeMetric, prometheus.GaugeValue, float64(fileSize))
	for qualityName, count := range qualities {
		ch <- prometheus.MustNewConstMetric(collector.qualitiesMetric, prometheus.GaugeValue, float64(count),
			qualityName,
		)
	}
}
