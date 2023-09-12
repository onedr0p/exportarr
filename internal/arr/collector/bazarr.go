package collector

import (
	"github.com/onedr0p/exportarr/internal/arr/client"
	"github.com/onedr0p/exportarr/internal/arr/config"
	"github.com/onedr0p/exportarr/internal/arr/model"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type bazarrCollector struct {
	config                    *config.ArrConfig // App configuration
	subtitleMetric            *prometheus.Desc  // Total number of subtitles
	subtitleDownloadedMetric  *prometheus.Desc  // Total number of subtitles downloaded
	subtitleMonitoredMetric   *prometheus.Desc  // Total number of subtitles monitored
	subtitleUnmonitoredMetric *prometheus.Desc  // Total number of subtitles unmonitored
	subtitleWantedMetric      *prometheus.Desc  // Total number of wanted subtitle
	subtitleMissingMetric     *prometheus.Desc  // Total number of missing subtitle
	subtitleFileSizeMetric    *prometheus.Desc  // Total fizesize of all subtitle in bytes <-- do we actually want this? maybe?
	subtitleLanguageMetric    *prometheus.Desc  // Total number of subtitle by language
	errorMetric               *prometheus.Desc  // Error Description for use with InvalidMetric
}

func NewBazarrCollector(c *config.ArrConfig) *bazarrCollector {
	return &bazarrCollector{
		config: c,
		subtitleMetric: prometheus.NewDesc(
			"bazarr_subtitle_total",
			"Total number of subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleDownloadedMetric: prometheus.NewDesc(
			"bazarr_subtitle_downloaded_total",
			"Total number of downloaded subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleMonitoredMetric: prometheus.NewDesc(
			"bazarr_subtitle_monitored_total",
			"Total number of monitored subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleUnmonitoredMetric: prometheus.NewDesc(
			"bazarr_subtitle_unmonitored_total",
			"Total number of unmonitored subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleWantedMetric: prometheus.NewDesc(
			"bazarr_subtitle_wanted_total",
			"Total number of wanted subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleMissingMetric: prometheus.NewDesc(
			"bazarr_subtitle_missing_total",
			"Total number of missing subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleFileSizeMetric: prometheus.NewDesc(
			"bazarr_subtitle_filesize_total",
			"Total filesize of all subtitles",
			nil,
			prometheus.Labels{"url": c.URL},
		),
		subtitleLanguageMetric: prometheus.NewDesc(
			"bazarr_language_total",
			"Total number of downloaded subtitles by quality",
			[]string{"quality"},
			prometheus.Labels{"url": c.URL},
		),
		errorMetric: prometheus.NewDesc(
			"bazarr_collector_error",
			"Error while collecting metrics",
			nil,
			prometheus.Labels{"url": c.URL},
		),
	}
}

func (collector *bazarrCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.subtitleMetric
	ch <- collector.subtitleDownloadedMetric
	ch <- collector.subtitleMonitoredMetric
	ch <- collector.subtitleUnmonitoredMetric
	ch <- collector.subtitleWantedMetric
	ch <- collector.subtitleMissingMetric
	ch <- collector.subtitleFileSizeMetric
	ch <- collector.subtitleLanguageMetric
}

func (collector *bazarrCollector) Collect(ch chan<- prometheus.Metric) {
	log := zap.S().With("collector", "bazarr")
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
		languages   = map[string]int{}
	)
	subtitles := model.Subtitle{}
	// https://radarr.video/docs/api/#/Subtitle/get_api_v3_subtitle
	if err := c.DoRequest("subtitle", &subtitles); err != nil {
		log.Errorw("Error getting subtitles", "error", err)
		ch <- prometheus.NewInvalidMetric(collector.errorMetric, err)
		return
	}
	for _, s := range subtitles {
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

		if s.SubtitleFile.Language.Name != "" {
			languages[s.SubtitleFile.Language.Name]++
		}
		if s.SubtitleFile.Size != 0 {
			fileSize += s.SubtitleFile.Size
		}
	}

	ch <- prometheus.MustNewConstMetric(collector.subtitleMetric, prometheus.GaugeValue, float64(len(subtitles)))
	ch <- prometheus.MustNewConstMetric(collector.subtitleDownloadedMetric, prometheus.GaugeValue, float64(downloaded))
	ch <- prometheus.MustNewConstMetric(collector.subtitleMonitoredMetric, prometheus.GaugeValue, float64(monitored))
	ch <- prometheus.MustNewConstMetric(collector.subtitleUnmonitoredMetric, prometheus.GaugeValue, float64(unmonitored))
	ch <- prometheus.MustNewConstMetric(collector.subtitleWantedMetric, prometheus.GaugeValue, float64(wanted))
	ch <- prometheus.MustNewConstMetric(collector.subtitleMissingMetric, prometheus.GaugeValue, float64(missing))
	ch <- prometheus.MustNewConstMetric(collector.subtitleFileSizeMetric, prometheus.GaugeValue, float64(fileSize))

	if len(languages) > 0 {
		for qualityName, count := range languages {
			ch <- prometheus.MustNewConstMetric(collector.subtitleLanguageMetric, prometheus.GaugeValue, float64(count),
				qualityName,
			)
		}
	}
}
