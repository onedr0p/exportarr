package collector

import (
	"fmt"
	"log/slog"
	"strconv"
	"sync"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type lidarrCollector struct {
	collectMu              sync.Mutex // Guards against overlapping collections (#380)
	client                 *client.Client
	config                 *config.ArrConfig // App configuration
	artistsMetric          *prometheus.Desc  // Total number of artists
	artistsMonitoredMetric *prometheus.Desc  // Total number of monitored artists
	artistGenresMetric     *prometheus.Desc  // Total number of artists by genre
	artistsFileSizeMetric  *prometheus.Desc  // Total fizesize of all artists in bytes
	albumsMetric           *prometheus.Desc  // Total number of albums
	albumsMonitoredMetric  *prometheus.Desc  // Total number of monitored albums
	albumsGenresMetric     *prometheus.Desc  // Total number of albums by genre
	albumsMissingMetric    *prometheus.Desc  // Total number of missing albums
	songsMetric            *prometheus.Desc  // Total number of songs
	songsMonitoredMetric   *prometheus.Desc  // Total number of monitored songs
	songsDownloadedMetric  *prometheus.Desc  // Total number of downloaded songs
	songsQualitiesMetric   *prometheus.Desc  // Total number of songs by quality
	errorMetric            *prometheus.Desc  // Error Description for use with InvalidMetric
}

// NewLidarrCollector builds a collector for lidarr library statistics.
func NewLidarrCollector(httpClient *client.Client, c *config.ArrConfig) prometheus.Collector {
	return &lidarrCollector{
		client:                 httpClient,
		config:                 c,
		artistsMetric:          newDesc("lidarr", "artists_total", "Total number of artists", nil, c.URL),
		artistsMonitoredMetric: newDesc("lidarr", "artists_monitored_total", "Total number of monitored artists", nil, c.URL),
		artistGenresMetric:     newDesc("lidarr", "artists_genres_total", "Total number of artists by genre", []string{"genre"}, c.URL),
		artistsFileSizeMetric:  newDesc("lidarr", "artists_filesize_bytes", "Total fizesize of all artists in bytes", nil, c.URL),
		albumsMetric:           newDesc("lidarr", "albums_total", "Total number of albums", nil, c.URL),
		albumsMonitoredMetric:  newDesc("lidarr", "albums_monitored_total", "Total number of albums", nil, c.URL),
		albumsGenresMetric:     newDesc("lidarr", "albums_genres_total", "Total number of albums by genre", []string{"genre"}, c.URL),
		albumsMissingMetric:    newDesc("lidarr", "albums_missing_total", "Total number of missing albums", nil, c.URL),
		songsMetric:            newDesc("lidarr", "songs_total", "Total number of songs", nil, c.URL),
		songsMonitoredMetric:   newDesc("lidarr", "songs_monitored_total", "Total number of monitored songs", nil, c.URL),
		songsDownloadedMetric:  newDesc("lidarr", "songs_downloaded_total", "Total number of downloaded songs", nil, c.URL),
		songsQualitiesMetric:   newDesc("lidarr", "songs_quality_total", "Total number of downloaded songs by quality", []string{"quality", "weight"}, c.URL),
		errorMetric:            newDesc("lidarr", "collector_error", "Error while collecting metrics", nil, c.URL),
	}
}

func (collector *lidarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.errorMetric
	ch <- collector.artistsMetric
	ch <- collector.artistsMonitoredMetric
	ch <- collector.artistGenresMetric
	ch <- collector.artistsFileSizeMetric
	ch <- collector.albumsMetric
	ch <- collector.albumsMonitoredMetric
	ch <- collector.albumsGenresMetric
	ch <- collector.albumsMissingMetric
	ch <- collector.songsMetric
	ch <- collector.songsMonitoredMetric
	ch <- collector.songsDownloadedMetric
	ch <- collector.songsQualitiesMetric
}

func (collector *lidarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := slog.With("collector", "lidarr")
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
	c := collector.client
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
		qualityWeights   = map[string]string{}
	)

	artists, err := client.Get[model.Artist](c, "artist")
	if err != nil {
		emitError(log, ch, collector.errorMetric, "Error creating client", "error", err)
		return
	}

	collectQuality := !collector.config.DisableQualityMetrics
	collectAlbums := !collector.config.DisableAlbumMetrics

	// Quality definitions are repository-global: fetch once, not per artist.
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
	}

	// The per-artist lookups dominate scrape time on large libraries: fan
	// them out with bounded concurrency instead of ~2×N serial requests.
	if collectQuality || collectAlbums {
		var mu sync.Mutex
		eg := errgroup.Group{}
		eg.SetLimit(maxConcurrentSeriesFetches)
		for _, s := range artists {
			goRecoverable(&eg, func() error {
				params := client.QueryParams{}
				params.Add("artistid", strconv.Itoa(s.ID))

				if collectQuality {
					songFile, err := client.Get[model.SongFile](c, "trackfile", params)
					if err != nil {
						return fmt.Errorf("getting trackfile for artist %d: %w", s.ID, err)
					}
					mu.Lock()
					for _, e := range songFile {
						if e.Quality.Quality.Name != "" {
							songsQualities[e.Quality.Quality.Name]++
						}
					}
					mu.Unlock()
				}
				if collectAlbums {
					album, err := client.Get[model.Album](c, "album", params)
					if err != nil {
						return fmt.Errorf("getting album for artist %d: %w", s.ID, err)
					}
					mu.Lock()
					for _, a := range album {
						if a.Monitored {
							albumsMonitored++
						}
						for _, genre := range s.Genres {
							albumGenres[genre]++
						}
					}
					mu.Unlock()
				}
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting per-artist metrics", "error", err)
			return
		}
	}

	// Only totalRecords is read: request the smallest page the API allows.
	// This total forces a full count, so it is skippable on huge instances.
	var albumsMissing int
	if !collector.config.DisableWantedMetrics {
		missingParams := client.QueryParams{}
		missingParams.Add("pageSize", "1")
		missing, err := client.Get[model.Missing](c, "wanted/missing", missingParams)
		if err != nil {
			emitError(log, ch, collector.errorMetric, "Error getting missing albums", "error", err)
			return
		}
		albumsMissing = missing.TotalRecords
	}

	ch <- prometheus.MustNewConstMetric(collector.artistsMetric, prometheus.GaugeValue, float64(len(artists)))
	ch <- prometheus.MustNewConstMetric(collector.artistsMonitoredMetric, prometheus.GaugeValue, float64(artistsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.artistsFileSizeMetric, prometheus.GaugeValue, float64(artistsFileSize))
	ch <- prometheus.MustNewConstMetric(collector.albumsMetric, prometheus.GaugeValue, float64(albums))
	if !collector.config.DisableWantedMetrics {
		ch <- prometheus.MustNewConstMetric(collector.albumsMissingMetric, prometheus.GaugeValue, float64(albumsMissing))
	}
	ch <- prometheus.MustNewConstMetric(collector.songsMetric, prometheus.GaugeValue, float64(songs))
	ch <- prometheus.MustNewConstMetric(collector.songsDownloadedMetric, prometheus.GaugeValue, float64(songsDownloaded))

	if len(artistGenres) > 0 {
		for genre, count := range artistGenres {
			ch <- prometheus.MustNewConstMetric(collector.artistGenresMetric, prometheus.GaugeValue, float64(count), genre)
		}
	}

	if collectAlbums {
		ch <- prometheus.MustNewConstMetric(collector.albumsMonitoredMetric, prometheus.GaugeValue, float64(albumsMonitored))

		if len(albumGenres) > 0 {
			for genre, count := range albumGenres {
				ch <- prometheus.MustNewConstMetric(collector.albumsGenresMetric, prometheus.GaugeValue, float64(count), genre)
			}
		}
	}
	if collectQuality && len(songsQualities) > 0 {
		for qualityName, count := range songsQualities {
			ch <- prometheus.MustNewConstMetric(collector.songsQualitiesMetric, prometheus.GaugeValue, float64(count), qualityName, qualityWeights[qualityName])
		}
	}
}
