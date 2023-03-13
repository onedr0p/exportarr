package collector

import (
	"time"

	"github.com/onedr0p/exportarr/internal/client"
	"github.com/onedr0p/exportarr/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

type readarrCollector struct {
	config                  *cli.Context     // App configuration
	configFile              *model.Config    // *arr configuration from config.xml
	authorMetric            *prometheus.Desc // Total number of authors
	authorDownloadedMetric  *prometheus.Desc // Total number of downloaded authors
	authorMonitoredMetric   *prometheus.Desc // Total number of monitored authors
	authorUnmonitoredMetric *prometheus.Desc // Total number of unmonitored authors
	authorFileSizeMetric    *prometheus.Desc // Total filesize of all authors in bytes
	bookMetric              *prometheus.Desc // Total number of monitored books
	bookGrabbedMetric       *prometheus.Desc // Total number of grabbed books
	bookDownloadedMetric    *prometheus.Desc // Total number of downloaded books
	bookMonitoredMetric     *prometheus.Desc // Total number of monitored books
	bookUnmonitoredMetric   *prometheus.Desc // Total number of unmonitored books
	bookMissingMetric       *prometheus.Desc // Total number of missing books
}

func NewReadarrCollector(c *cli.Context, cf *model.Config) *readarrCollector {
	return &readarrCollector{
		config:     c,
		configFile: cf,
		authorMetric: prometheus.NewDesc(
			"readarr_author_total",
			"Total number of authors",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		authorDownloadedMetric: prometheus.NewDesc(
			"readarr_author_downloaded_total",
			"Total number of downloaded authors",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		authorMonitoredMetric: prometheus.NewDesc(
			"readarr_author_monitored_total",
			"Total number of monitored authors",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		authorUnmonitoredMetric: prometheus.NewDesc(
			"readarr_author_unmonitored_total",
			"Total number of unmonitored authors",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		authorFileSizeMetric: prometheus.NewDesc(
			"readarr_author_filesize_bytes",
			"Total filesize of all authors in bytes",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookMetric: prometheus.NewDesc(
			"readarr_book_total",
			"Total number of books",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookGrabbedMetric: prometheus.NewDesc(
			"readarr_book_grabbed_total",
			"Total number of grabbed books",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookDownloadedMetric: prometheus.NewDesc(
			"readarr_book_downloaded_total",
			"Total number of downloaded books",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookMonitoredMetric: prometheus.NewDesc(
			"readarr_book_monitored_total",
			"Total number of monitored books",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookUnmonitoredMetric: prometheus.NewDesc(
			"readarr_book_unmonitored_total",
			"Total number of unmonitored books",
			nil,
			prometheus.Labels{"url": c.String("url")},
		),
		bookMissingMetric: prometheus.NewDesc(
			"readarr_book_missing_total",
			"Total number of missing books",
			nil,
			prometheus.Labels{"url": c.String("url")},
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
	ch <- c.bookDownloadedMetric
	ch <- c.bookMonitoredMetric
	ch <- c.bookUnmonitoredMetric
	ch <- c.bookMissingMetric
}

func (collector *readarrCollector) Collect(ch chan<- prometheus.Metric) {
	total := time.Now()
	c, err := client.NewClient(collector.config, collector.configFile)
	if err != nil {
		log.Errorf("Error creating client: %w", err)
		ch <- prometheus.NewInvalidMetric(
			prometheus.NewDesc(
				"readarr_collector_error",
				"Error Collecting from Readarr",
				nil,
				prometheus.Labels{"url": collector.config.String("url")}),
			err)
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
		log.Fatal(err)
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
		log.Debug("TIME :: author %s took %s", a.AuthorName, b)
	}

	books := model.Book{}
	if err := c.DoRequest("book", &books); err != nil {
		log.Fatal(err)
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

	log.Debug("TIME :: total took %s with author timings as %s",
		time.Since(total),
		tauthors,
	)
}
