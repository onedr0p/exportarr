package collector

import (
	"fmt"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type lidarrCollector struct {
	config                 *cli.Context     // App configuration
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
}

func NewLidarrCollector(c *cli.Context) *lidarrCollector {
	return &lidarrCollector{
		config: c,
		artistsMetric: prometheus.NewDesc(
			"lidarr_artists_total",
			"Total number of artists",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		artistsMonitoredMetric: prometheus.NewDesc(
			"lidarr_artists_monitored_total",
			"Total number of monitored artists",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		artistGenresMetric: prometheus.NewDesc(
			"lidarr_artists_genres_total",
			"Total number of artists by genre",
			[]string{"genre"},
			prometheus.Labels{"url": c.String("url")},
		),
		artistsFileSizeMetric: prometheus.NewDesc(
			"lidarr_artists_filesize_bytes",
			"Total fizesize of all artists in bytes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		albumsMetric: prometheus.NewDesc(
			"lidarr_albums_total",
			"Total number of albums",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		albumsMonitoredMetric: prometheus.NewDesc(
			"lidarr_albums_monitored_total",
			"Total number of albums",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		albumsGenresMetric: prometheus.NewDesc(
			"lidarr_albums_genres_total",
			"Total number of albums by genre",
			[]string{"genre"},
			prometheus.Labels{"url": c.String("url")},
		),
		songsMetric: prometheus.NewDesc(
			"lidarr_songs_total",
			"Total number of songs",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		songsMonitoredMetric: prometheus.NewDesc(
			"lidarr_songs_monitored_total",
			"Total number of monitored songs",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		songsDownloadedMetric: prometheus.NewDesc(
			"lidarr_songs_downloaded_total",
			"Total number of downloaded songs",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		songsMissingMetric: prometheus.NewDesc(
			"lidarr_songs_missing_total",
			"Total number of missing songs",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		songsQualitiesMetric: prometheus.NewDesc(
			"lidarr_songs_quality_total",
			"Total number of downloaded songs by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.String("url")},
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
	c := client.NewClient(collector.config)

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
		log.Fatal(err)
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

		if collector.config.Bool("enable-song-quality-metrics") {
			songFile := model.SongFile{}
			if err := c.DoRequest(fmt.Sprintf("%s?artistid=%d", "trackfile", s.Id), &songFile); err != nil {
				log.Fatal(err)
			}
			for _, e := range songFile {
				if e.Quality.Quality.Name != "" {
					songsQualities[e.Quality.Quality.Name]++
				}
			}
		}

		if collector.config.Bool("enable-monitored-albums-metrics") {
			album := model.Album{}
			if err := c.DoRequest(fmt.Sprintf("%s?artistid=%d", "album", s.Id), &album); err != nil {
				log.Fatal(err)
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
	if err := c.DoRequest("wanted/missing?sortKey=releaseDate", &songMissing); err != nil {
		log.Fatal(err)
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

	if collector.config.Bool("enable-song-quality-metrics") {
		if len(songsQualities) > 0 {
			for qualityName, count := range songsQualities {
				ch <- prometheus.MustNewConstMetric(collector.songsQualitiesMetric, prometheus.GaugeValue, float64(count), qualityName)
			}
		}
	}

	if collector.config.Bool("enable-monitored-albums-metrics") {
		ch <- prometheus.MustNewConstMetric(collector.albumsMonitoredMetric, prometheus.GaugeValue, float64(albumsMonitored))

		if len(albumGenres) > 0 {
			for genre, count := range albumGenres {
				ch <- prometheus.MustNewConstMetric(collector.albumsGenresMetric, prometheus.GaugeValue, float64(count), genre)
			}
		}
	}
}
