package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type lidarrCollector struct {
	config                 *config.Config   // App configuration
	artistsMetric          *prometheus.Desc // Total number of artists
	artistsMonitoredMetric *prometheus.Desc // Total number of monitored artists
	artistGenresMetric     *prometheus.Desc // Total number of artists by genre
	artistsFileSizeMetric  *prometheus.Desc // Total fizesize of all artists in bytes
	albumsMetric           *prometheus.Desc // Total number of albums
	albumsMonitoredMetric  *prometheus.Desc // Total number of monitored albums
	albumsGenresMetric     *prometheus.Desc // Total number of albums by genre
	songsMetric            *prometheus.Desc // Total number of songs
	songsMonitoredMetric   *prometheus.Desc // Total number of monitored songs
	songsDownloadedMetric  *prometheus.Desc // Total number of downloaded songs
	songsMissingMetric     *prometheus.Desc // Total number of missing songs
	songsQualitiesMetric   *prometheus.Desc // Total number of songs by quality
	errorMetric            *prometheus.Desc // Error Description for use with InvalidMetric
}

func NewLidarrCollector(c *config.Config) *lidarrCollector {
	return &lidarrCollector{
		config: c,
		artistsMetric: prometheus.NewDesc(
			"lidarr_artists_total",
			"Total number of artists",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		artistsMonitoredMetric: prometheus.NewDesc(
			"lidarr_artists_monitored_total",
			"Total number of monitored artists",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		artistGenresMetric: prometheus.NewDesc(
			"lidarr_artists_genres_total",
			"Total number of artists by genre",
			[]string{"genre"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		artistsFileSizeMetric: prometheus.NewDesc(
			"lidarr_artists_filesize_bytes",
			"Total fizesize of all artists in bytes",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		albumsMetric: prometheus.NewDesc(
			"lidarr_albums_total",
			"Total number of albums",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		albumsMonitoredMetric: prometheus.NewDesc(
			"lidarr_albums_monitored_total",
			"Total number of albums",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		albumsGenresMetric: prometheus.NewDesc(
			"lidarr_albums_genres_total",
			"Total number of albums by genre",
			[]string{"genre"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		songsMetric: prometheus.NewDesc(
			"lidarr_songs_total",
			"Total number of songs",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		songsMonitoredMetric: prometheus.NewDesc(
			"lidarr_songs_monitored_total",
			"Total number of monitored songs",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		songsDownloadedMetric: prometheus.NewDesc(
			"lidarr_songs_downloaded_total",
			"Total number of downloaded songs",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		songsMissingMetric: prometheus.NewDesc(
			"lidarr_songs_missing_total",
			"Total number of missing songs",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
		songsQualitiesMetric: prometheus.NewDesc(
			"lidarr_songs_quality_total",
			"Total number of downloaded songs by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.URLLabel()},
		),
		errorMetric: prometheus.NewDesc(
			"lidarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URLLabel()},
		),
	}
}

func (collector *lidarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.artistsMetric
	ch <- collector.artistsMonitoredMetric
	ch <- collector.artistGenresMetric
	ch <- collector.artistsFileSizeMetric
	ch <- collector.albumsMetric
	ch <- collector.albumsMonitoredMetric
	ch <- collector.albumsGenresMetric
	ch <- collector.songsMetric
	ch <- collector.songsMonitoredMetric
	ch <- collector.songsDownloadedMetric
	ch <- collector.songsMissingMetric
	ch <- collector.songsQualitiesMetric
}

func (collector *lidarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "lidarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorf("Error creating client", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	var artistsFileSize int64
	var (
		artistsMonitored = 0
		artistGenres     = map[string]int{}
		albums           = 0
		albumsMonitored  = 0
		albumGenres      = map[string]int{}
		songs            = 0
		songsDownloaded  = 0
		songsQualities   = map[string]int{}
	)

	artists := model.Artist{}
	if err := c.DoRequest("artist", &artists); err != nil {
		log.Errorw("Error creating client", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	for _, s := range artists {
		if s.Monitored {
			artistsMonitored++
		}
		albums += s.Statistics.AlbumCount
		songs += s.Statistics.TotalTrackCount
		songsDownloaded += s.Statistics.TrackFileCount
		artistsFileSize += s.Statistics.SizeOnDisk

		for _, genre := range s.Genres {
			artistGenres[genre]++
		}

		if collector.config.EnableAdditionalMetrics {
			songFile := model.SongFile{}
			params := map[string]string{"artistid": fmt.Sprintf("%d", s.Id)}
			if err := c.DoRequest("trackfile", &songFile, params); err != nil {
				log.Errorw("Error getting trackfile", "error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			for _, e := range songFile {
				if e.Quality.Quality.Name != "" {
					songsQualities[e.Quality.Quality.Name]++
				}
			}

			album := model.Album{}
			params = map[string]string{"artistid": fmt.Sprintf("%d", s.Id)}
			if err := c.DoRequest("album", &album, params); err != nil {
				log.Errorw("Error getting album", "error", err)
				ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
				return
			}
			for _, a := range album {
				if a.Monitored {
					albumsMonitored++
				}
				for _, genre := range s.Genres {
					albumGenres[genre]++
				}
			}
		}
	}

	songMissing := model.Missing{}
	if err := c.DoRequest("wanted/missing", &songMissing); err != nil {
		log.Errorw("Error getting missing", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(collector.artistsMetric, prometheus.GaugeValue, float64(len(artists)))
	ch <- prometheus.MustNewConstMetric(collector.artistsMonitoredMetric, prometheus.GaugeValue, float64(artistsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.artistsFileSizeMetric, prometheus.GaugeValue, float64(artistsFileSize))
	ch <- prometheus.MustNewConstMetric(collector.albumsMetric, prometheus.GaugeValue, float64(albums))
	ch <- prometheus.MustNewConstMetric(collector.songsMetric, prometheus.GaugeValue, float64(songs))
	ch <- prometheus.MustNewConstMetric(collector.songsDownloadedMetric, prometheus.GaugeValue, float64(songsDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.songsMissingMetric, prometheus.GaugeValue, float64(songMissing.TotalRecords))

	if len(artistGenres) > 0 {
		for genre, count := range artistGenres {
			ch <- prometheus.MustNewConstMetric(collector.artistGenresMetric, prometheus.GaugeValue, float64(count), genre)
		}
	}

	if collector.config.EnableAdditionalMetrics {
		ch <- prometheus.MustNewConstMetric(collector.albumsMonitoredMetric, prometheus.GaugeValue, float64(albumsMonitored))

		if len(songsQualities) > 0 {
			for qualityName, count := range songsQualities {
				ch <- prometheus.MustNewConstMetric(collector.songsQualitiesMetric, prometheus.GaugeValue, float64(count), qualityName)
			}
		}

		if len(albumGenres) > 0 {
			for genre, count := range albumGenres {
				ch <- prometheus.MustNewConstMetric(collector.albumsGenresMetric, prometheus.GaugeValue, float64(count), genre)
			}
		}
	}
}
