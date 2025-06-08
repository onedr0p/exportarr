package collector

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shamelin/exportarr/internal/arr/client"
	"github.com/shamelin/exportarr/internal/arr/config"
	"github.com/shamelin/exportarr/internal/arr/model"
	"go.uber.org/zap"
)

type radarrCollector struct {
	config                 *config.ArrConfig // App configuration
	movieEdition           *prometheus.Desc  // Total number of movies with an `edition` set
	movieMetric            *prometheus.Desc  // Total number of movies
	movieDownloadedMetric  *prometheus.Desc  // Total number of downloaded movies
	movieMonitoredMetric   *prometheus.Desc  // Total number of monitored movies
	movieUnmonitoredMetric *prometheus.Desc  // Total number of unmonitored movies
	movieWantedMetric      *prometheus.Desc  // Total number of wanted movies
	movieMissingMetric     *prometheus.Desc  // Total number of missing movies
	movieQualitiesMetric   *prometheus.Desc  // Total number of movies by quality
	movieFileSizeMetric    *prometheus.Desc  // Total fizesize of all movies in bytes
	errorMetric            *prometheus.Desc  // Error Description for use with InvalidMetric
	movieTagsMetric        *prometheus.Desc  // Total number of downloaded movies by tag
}

func NewRadarrCollector(c *config.ArrConfig) *radarrCollector {
	return &radarrCollector{
		config: c,
		movieEdition: prometheus.NewDesc(
			"radarr_movie_editions",
			"Total number of movies with `edition` set",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieMetric: prometheus.NewDesc(
			"radarr_movie_total",
			"Total number of movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieDownloadedMetric: prometheus.NewDesc(
			"radarr_movie_downloaded_total",
			"Total number of downloaded movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieMonitoredMetric: prometheus.NewDesc(
			"radarr_movie_monitored_total",
			"Total number of monitored movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieUnmonitoredMetric: prometheus.NewDesc(
			"radarr_movie_unmonitored_total",
			"Total number of unmonitored movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieWantedMetric: prometheus.NewDesc(
			"radarr_movie_wanted_total",
			"Total number of wanted movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieMissingMetric: prometheus.NewDesc(
			"radarr_movie_missing_total",
			"Total number of missing movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieFileSizeMetric: prometheus.NewDesc(
			"radarr_movie_filesize_total",
			"Total filesize of all movies",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		movieQualitiesMetric: prometheus.NewDesc(
			"radarr_movie_quality_total",
			"Total number of downloaded movies by quality",
			[]string{"quality", "weight"},
			prometheus.Labels{"url": c.URL},
		),
		movieTagsMetric: prometheus.NewDesc(
			"radarr_movie_tag_total",
			"Total number of downloaded movies by tag",
			[]string{"tag"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			"radarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *radarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.movieEdition
	ch <- collector.movieMetric
	ch <- collector.movieDownloadedMetric
	ch <- collector.movieMonitoredMetric
	ch <- collector.movieUnmonitoredMetric
	ch <- collector.movieWantedMetric
	ch <- collector.movieMissingMetric
	ch <- collector.movieFileSizeMetric
	ch <- collector.movieQualitiesMetric
	ch <- collector.movieTagsMetric
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
		editions    = 0
		downloaded  = 0
		monitored   = 0
		unmonitored = 0
		missing     = 0
		wanted      = 0
		qualities   = map[string]int{}
		tags        = []struct {
			Label  string
			Movies int
		}{}
		qualityWeights = map[string]string{}
	)

	movies := model.Movie{}
	params := client.QueryParams{}
	params.Add("excludeLocalCovers", "true")

	// https://radarr.video/docs/api/#/Movie/get_api_v3_movie
	if err := c.DoRequest("movie", &movies, params); err != nil {
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

		if s.MovieFile.Edition != "" {
			editions++
		}
	}

	tagObjects := model.TagMovies{}
	// https://radarr.video/docs/api/#/TagDetails/get_api_v3_tag_detail
	if err := c.DoRequest("tag/detail", &tagObjects); err != nil {
		log.Errorw("Error getting Tags", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, s := range tagObjects {
		tag := struct {
			Label  string
			Movies int
		}{
			Label:  s.Label,
			Movies: len(s.MovieIds),
		}
		tags = append(tags, tag)
	}

	qualityDefs := model.Qualities{}
	if err := c.DoRequest("qualitydefinition", &qualityDefs); err != nil {
		log.Errorw("Error getting qualities",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, q := range qualityDefs {
		if q.Quality.Name != "" {
			qualityWeights[q.Quality.Name] = strconv.Itoa(q.Weight)
		}
	}

	ch <- prometheus.MustNewConstMetric(collector.movieEdition, prometheus.GaugeValue, float64(editions))
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
				qualityName, qualityWeights[qualityName],
			)
		}
	}

	if len(tags) > 0 {
		for _, Tag := range tags {
			ch <- prometheus.MustNewConstMetric(collector.movieTagsMetric, prometheus.GaugeValue, float64(Tag.Movies),
				Tag.Label,
			)
		}
	}

}
