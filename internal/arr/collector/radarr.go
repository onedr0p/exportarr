package collector

import (
	"log/slog"
	"strconv"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
)

type radarrCollector struct {
	client                 *client.Client
	config                 *config.ArrConfig // App configuration
	movieEdition           *prometheus.Desc  // Total number of movies with an `edition` set
	movieMetric            *prometheus.Desc  // Total number of movies
	movieDownloadedMetric  *prometheus.Desc  // Total number of downloaded movies
	movieMonitoredMetric   *prometheus.Desc  // Total number of monitored movies
	movieUnmonitoredMetric *prometheus.Desc  // Total number of unmonitored movies
	movieWantedMetric      *prometheus.Desc  // Total number of wanted movies
	movieMissingMetric     *prometheus.Desc  // Total number of missing movies
	movieCutoffUnmetMetric *prometheus.Desc  // Total number of movies with cutoff unmet
	movieQualitiesMetric   *prometheus.Desc  // Total number of movies by quality
	movieFileSizeMetric    *prometheus.Desc  // Total fizesize of all movies in bytes
	errorMetric            *prometheus.Desc  // Error Description for use with InvalidMetric
	movieTagsMetric        *prometheus.Desc  // Total number of downloaded movies by tag
}

// NewRadarrCollector builds a collector for radarr library statistics.
func NewRadarrCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &radarrCollector{
		client:                 httpClient,
		config:                 c,
		movieEdition:           newDesc("radarr", "movie_editions", "Total number of movies with `edition` set", nil, c.URL),
		movieMetric:            newDesc("radarr", "movie_total", "Total number of movies", nil, c.URL),
		movieDownloadedMetric:  newDesc("radarr", "movie_downloaded_total", "Total number of downloaded movies", nil, c.URL),
		movieMonitoredMetric:   newDesc("radarr", "movie_monitored_total", "Total number of monitored movies", nil, c.URL),
		movieUnmonitoredMetric: newDesc("radarr", "movie_unmonitored_total", "Total number of unmonitored movies", nil, c.URL),
		movieWantedMetric:      newDesc("radarr", "movie_wanted_total", "Total number of wanted movies", nil, c.URL),
		movieMissingMetric:     newDesc("radarr", "movie_missing_total", "Total number of missing movies", nil, c.URL),
		movieCutoffUnmetMetric: newDesc("radarr", "movie_cutoff_unmet_total", "Total number of movies with cutoff unmet", nil, c.URL),
		movieFileSizeMetric:    newDesc("radarr", "movie_filesize_total", "Total filesize of all movies", nil, c.URL),
		movieQualitiesMetric:   newDesc("radarr", "movie_quality_total", "Total number of downloaded movies by quality", []string{"quality", "weight"}, c.URL),
		movieTagsMetric:        newDesc("radarr", "movie_tag_total", "Total number of downloaded movies by tag", []string{"tag"}, c.URL),
		errorMetric:            newDesc("radarr", "collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *radarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.movieEdition
	ch <- collector.movieMetric
	ch <- collector.movieDownloadedMetric
	ch <- collector.movieMonitoredMetric
	ch <- collector.movieUnmonitoredMetric
	ch <- collector.movieWantedMetric
	ch <- collector.movieMissingMetric
	ch <- collector.movieCutoffUnmetMetric
	ch <- collector.movieFileSizeMetric
	ch <- collector.movieQualitiesMetric
	ch <- collector.movieTagsMetric
}

func (collector *radarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "radarr")
	defer recoverCollect(log, ch, collector.errorMetric)
	c := collector.client
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

	params := client.QueryParams{}
	params.Add("excludeLocalCovers", "true")

	// https://radarr.video/docs/api/#/Movie/get_api_v3_movie
	movies, err := client.Get[model.Movie](c, "movie", params)
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting movies", "error", err)
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

	// https://radarr.video/docs/api/#/TagDetails/get_api_v3_tag_detail
	tagObjects, err := client.Get[model.TagMovies](c, "tag/detail")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting Tags", "error", err)
		return
	}
	for _, s := range tagObjects {
		tag := struct {
			Label  string
			Movies int
		}{
			Label:  s.Label,
			Movies: len(s.MovieIDs),
		}
		tags = append(tags, tag)
	}

	qualityDefs, err := client.Get[model.Qualities](c, "qualitydefinition")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error getting qualities", "error", err)
		return
	}
	for _, q := range qualityDefs {
		if q.Quality.Name != "" {
			qualityWeights[q.Quality.Name] = strconv.Itoa(q.Weight)
		}
	}

	// Only totalRecords is read: request the smallest page the API allows.
	// This total forces a full count, so it is skippable on huge instances.
	var moviesCutoffUnmet int
	if !collector.config.DisableWantedMetrics {
		cutoffParams := client.QueryParams{}
		cutoffParams.Add("pageSize", "1")
		cutoffUnmet, err := client.Get[model.CutoffUnmetMovies](c, "wanted/cutoff", cutoffParams)
		if err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting cutoff unmet", "error", err)
			return
		}
		moviesCutoffUnmet = cutoffUnmet.TotalRecords
	}

	ch <- prometheus.MustNewConstMetric(collector.movieEdition, prometheus.GaugeValue, float64(editions))
	ch <- prometheus.MustNewConstMetric(collector.movieMetric, prometheus.GaugeValue, float64(len(movies)))
	ch <- prometheus.MustNewConstMetric(collector.movieDownloadedMetric, prometheus.GaugeValue, float64(downloaded))
	ch <- prometheus.MustNewConstMetric(collector.movieMonitoredMetric, prometheus.GaugeValue, float64(monitored))
	ch <- prometheus.MustNewConstMetric(collector.movieUnmonitoredMetric, prometheus.GaugeValue, float64(unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.movieWantedMetric, prometheus.GaugeValue, float64(wanted))
	ch <- prometheus.MustNewConstMetric(collector.movieMissingMetric, prometheus.GaugeValue, float64(missing))
	if !collector.config.DisableWantedMetrics {
		ch <- prometheus.MustNewConstMetric(collector.movieCutoffUnmetMetric, prometheus.GaugeValue, float64(moviesCutoffUnmet))
	}
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
