package collector

import (
	"time"

	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type readarrCollector struct {
	config                  *config.ArrConfig // App configuration
	authorMetric            *prometheus.Desc  // Total number of authors
	authorDownloadedMetric  *prometheus.Desc  // Total number of downloaded authors
	authorMonitoredMetric   *prometheus.Desc  // Total number of monitored authors
	authorUnmonitoredMetric *prometheus.Desc  // Total number of unmonitored authors
	authorFileSizeMetric    *prometheus.Desc  // Total filesize of all authors in bytes
	bookMetric              *prometheus.Desc  // Total number of monitored books
	bookGrabbedMetric       *prometheus.Desc  // Total number of grabbed books
	bookDownloadedMetric    *prometheus.Desc  // Total number of downloaded books
	bookMonitoredMetric     *prometheus.Desc  // Total number of monitored books
	bookUnmonitoredMetric   *prometheus.Desc  // Total number of unmonitored books
	bookMissingMetric       *prometheus.Desc  // Total number of missing books
	errorMetric             *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewReadarrCollector(c *config.ArrConfig) *readarrCollector {
	return &readarrCollector{
		config: c,
		authorMetric: prometheus.NewDesc(
			"readarr_author_total",
			"Total number of authors",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		authorDownloadedMetric: prometheus.NewDesc(
			"readarr_author_downloaded_total",
			"Total number of downloaded authors",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		authorMonitoredMetric: prometheus.NewDesc(
			"readarr_author_monitored_total",
			"Total number of monitored authors",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		authorUnmonitoredMetric: prometheus.NewDesc(
			"readarr_author_unmonitored_total",
			"Total number of unmonitored authors",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		authorFileSizeMetric: prometheus.NewDesc(
			"readarr_author_filesize_bytes",
			"Total filesize of all authors in bytes",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookMetric: prometheus.NewDesc(
			"readarr_book_total",
			"Total number of books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookGrabbedMetric: prometheus.NewDesc(
			"readarr_book_grabbed_total",
			"Total number of grabbed books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookDownloadedMetric: prometheus.NewDesc(
			"readarr_book_downloaded_total",
			"Total number of downloaded books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookMonitoredMetric: prometheus.NewDesc(
			"readarr_book_monitored_total",
			"Total number of monitored books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookUnmonitoredMetric: prometheus.NewDesc(
			"readarr_book_unmonitored_total",
			"Total number of unmonitored books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		bookMissingMetric: prometheus.NewDesc(
			"readarr_book_missing_total",
			"Total number of missing books",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			"readarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (c *readarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.authorMetric
	ch <- c.authorDownloadedMetric
	ch <- c.authorMonitoredMetric
	ch <- c.authorUnmonitoredMetric
	ch <- c.authorFileSizeMetric
	ch <- c.bookMetric
	ch <- c.bookGrabbedMetric
	ch <- c.bookDownloadedMetric
	ch <- c.bookMonitoredMetric
	ch <- c.bookUnmonitoredMetric
	ch <- c.bookMissingMetric
}

func (collector *readarrCollector) Collect(ch chan<- prometheus.Metric) {
	total := time.Now()
	log := zap.S().With("collector", "readarr")
	c, err := client.NewClient(collector.config)
	if err != nil {
		log.Errorw("Error creating client", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	tauthors := []time.Duration{}
	var authorsFileSize int64
	var (
		authorsDownloaded  = 0
		authorsMonitored   = 0
		authorsUnmonitored = 0
		bookCount          = 0
		booksDownloaded    = 0
		booksMonitored     = 0
		booksUnmonitored   = 0
		booksGrabbed       = 0
		booksMissing       = 0
	)

	authors := model.Author{}
	if err := c.DoRequest("author", &authors); err != nil {
		log.Errorw("Error getting authors", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}

	for _, a := range authors {
		tauthor := time.Now()

		if !a.Monitored {
			authorsUnmonitored++
		} else {
			authorsMonitored++
		}

		bookCount += a.Statistics.BookCount
		booksDownloaded += a.Statistics.BookFileCount
		authorsFileSize += a.Statistics.SizeOnDisk

		if a.Statistics.PercentOfBooks == 100 {
			authorsDownloaded++
		}
		b := time.Since(tauthor)
		tauthors = append(tauthors, b)
		log.Debugw("author metrics retrieved",
			"author", a.AuthorName,
			"duration", b)
	}

	books := model.Book{}
	if err := c.DoRequest("book", &books); err != nil {
		log.Errorw("Error getting books",
			"error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, b := range books {
		if !b.Monitored {
			booksUnmonitored++
		} else {
			booksMonitored++
		}
		if b.Grabbed {
			booksGrabbed++
		}

		if b.Monitored && b.Statistics.BookFileCount == 0 {
			booksMissing++
		}
	}
	ch <- prometheus.MustNewConstMetric(collector.authorMetric, prometheus.GaugeValue, float64(len(authors)))
	ch <- prometheus.MustNewConstMetric(collector.authorDownloadedMetric, prometheus.GaugeValue, float64(authorsDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.authorMonitoredMetric, prometheus.GaugeValue, float64(authorsMonitored))
	ch <- prometheus.MustNewConstMetric(collector.authorUnmonitoredMetric, prometheus.GaugeValue, float64(authorsUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.authorFileSizeMetric, prometheus.GaugeValue, float64(authorsFileSize))
	ch <- prometheus.MustNewConstMetric(collector.bookMetric, prometheus.GaugeValue, float64(bookCount))
	ch <- prometheus.MustNewConstMetric(collector.bookGrabbedMetric, prometheus.GaugeValue, float64(booksGrabbed))
	ch <- prometheus.MustNewConstMetric(collector.bookDownloadedMetric, prometheus.GaugeValue, float64(booksDownloaded))
	ch <- prometheus.MustNewConstMetric(collector.bookMonitoredMetric, prometheus.GaugeValue, float64(booksMonitored))
	ch <- prometheus.MustNewConstMetric(collector.bookUnmonitoredMetric, prometheus.GaugeValue, float64(booksUnmonitored))
	ch <- prometheus.MustNewConstMetric(collector.bookMissingMetric, prometheus.GaugeValue, float64(booksMissing))

	log.Debugf("collector cycle completed",
		"duration", time.Since(total),
		"author_durations", tauthors,
	)
}
