package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	lidarrCollector "github.com/onedr0p/exportarr/internal/collector/lidarr"
	radarrCollector "github.com/onedr0p/exportarr/internal/collector/radarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
	sonarrCollector "github.com/onedr0p/exportarr/internal/collector/sonarr"
	"github.com/onedr0p/exportarr/internal/handlers"
	"github.com/onedr0p/exportarr/internal/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "Exportarr"
	app.Usage = "AIO Prometheus Exporter for Sonarr, Radarr or Lidarr"
	app.EnableBashCompletion = true
	app.HideVersion = true
	app.Authors = []*cli.Author{
		&cli.Author{
			Name:  "onedr0p",
			Email: "onedr0p@users.noreply.github.com",
		},
	}
	// Global flags
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:     "log-level",
			Aliases:  []string{"l"},
			Usage:    "Set the default Log Level",
			Value:    "INFO",
			Required: false,
			EnvVars:  []string{"LOG_LEVEL"},
		},
	}
	app.Before = func(c *cli.Context) error {
		switch strings.ToUpper(c.String("log-level")) {
		case "TRACE":
			log.SetLevel(log.TraceLevel)
		case "DEBUG":
			log.SetLevel(log.DebugLevel)
		case "INFO":
			log.SetLevel(log.InfoLevel)
		case "WARN":
			log.SetLevel(log.WarnLevel)
		default:
			log.SetLevel(log.TraceLevel)
		}
		log.SetFormatter(&log.TextFormatter{})
		log.SetOutput(os.Stdout)
		return nil
	}
	app.Commands = []*cli.Command{
		{
			Name:        "radarr",
			Aliases:     []string{"r"},
			Usage:       "Use the exporter for Radarr",
			Description: strings.Title("Radarr Exporter"),
			Flags:       flags("radarr"),
			Action:      radarr,
			Before:      validation,
		},
		{
			Name:        "sonarr",
			Aliases:     []string{"s"},
			Usage:       "Use the exporter for Sonarr",
			Description: strings.Title("Sonarr Exporter"),
			Flags:       flags("sonarr"),
			Action:      sonarr,
			Before:      validation,
		},
		{
			Name:        "lidarr",
			Aliases:     []string{"s"},
			Usage:       "Use the exporter for Lidarr",
			Description: strings.Title("Lidarr Exporter"),
			Flags:       flags("lidarr"),
			Action:      lidarr,
			Before:      validation,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func radarr(c *cli.Context) (err error) {
	r := prometheus.NewRegistry()
	r.MustRegister(
		radarrCollector.NewRadarrCollector(c),
		sharedCollector.NewQueueCollector(c),
		sharedCollector.NewHistoryCollector(c),
		sharedCollector.NewRootFolderCollector(c),
		sharedCollector.NewSystemStatusCollector(c),
		sharedCollector.NewSystemHealthCollector(c),
	)
	return serveHttp(c, r)
}

func sonarr(c *cli.Context) (err error) {
	r := prometheus.NewRegistry()
	r.MustRegister(
		sonarrCollector.NewSonarrCollector(c),
		sharedCollector.NewQueueCollector(c),
		sharedCollector.NewHistoryCollector(c),
		sharedCollector.NewRootFolderCollector(c),
		sharedCollector.NewSystemStatusCollector(c),
		sharedCollector.NewSystemHealthCollector(c),
	)
	return serveHttp(c, r)
}

func lidarr(c *cli.Context) (err error) {
	r := prometheus.NewRegistry()
	r.MustRegister(
		lidarrCollector.NewLidarrCollector(c),
		sharedCollector.NewQueueCollector(c),
		sharedCollector.NewHistoryCollector(c),
		sharedCollector.NewRootFolderCollector(c),
		sharedCollector.NewSystemStatusCollector(c),
		sharedCollector.NewSystemHealthCollector(c),
	)
	return serveHttp(c, r)
}

func serveHttp(c *cli.Context, r *prometheus.Registry) error {
	// Set up the handlers
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/healthz", handlers.HealthzHandler)
	http.Handle("/metrics", handler)
	// Serve up the metrics
	log.Infof("Listening on %s:%d", c.String("interface"), c.Int("port"))
	httpErr := http.ListenAndServe(
		fmt.Sprintf("%s:%d", c.String("interface"), c.Int("port")),
		logRequest(http.DefaultServeMux),
	)
	if httpErr != nil {
		return httpErr
	}
	return nil
}

// Log internal request to stdout
func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

// Validation used for all services
func validation(c *cli.Context) error {
	if !utils.IsValidUrl(c.String("url")) {
		return cli.Exit(fmt.Sprintf("%s is not a valid URL", c.String("url")), 10)
	}
	if !utils.IsValidApikey(c.String("api-key")) {
		return cli.Exit(fmt.Sprintf("%s is not a valid API Key", c.String("api-key")), 11)
	}
	return nil
}

// Flags used for all services
func flags(whatarr string) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Aliases:  []string{"u"},
			Usage:    fmt.Sprintf("%s's full URL", whatarr),
			Required: true,
			EnvVars:  []string{"URL"},
		},
		&cli.StringFlag{
			Name:     "api-key",
			Aliases:  []string{"a"},
			Usage:    fmt.Sprintf("%s's API Key", whatarr),
			Required: true,
			EnvVars:  []string{"APIKEY"},
		},
		&cli.IntFlag{
			Name:     "port",
			Aliases:  []string{"p"},
			Usage:    "Port the exporter will listen on",
			Required: true,
			EnvVars:  []string{"PORT"},
		},
		&cli.StringFlag{
			Name:     "interface",
			Aliases:  []string{"i"},
			Usage:    "IP the exporter will listen on",
			Value:    "0.0.0.0",
			Required: false,
			EnvVars:  []string{"INTERFACE"},
		},
		&cli.BoolFlag{
			Name:     "disable-ssl-verify",
			Usage:    "Disable SSL Verifications (use with caution)",
			Value:    false,
			Required: false,
			EnvVars:  []string{"DISABLE_SSL_VERIFY"},
		},
		&cli.BoolFlag{
			Name:     "basic-auth-enabled",
			Usage:    "Enable Basic Auth",
			Value:    false,
			Required: false,
			EnvVars:  []string{"BASIC_AUTH_ENABLED"},
		},
		&cli.StringFlag{
			Name:     "basic-auth-username",
			Usage:    "If Basic Auth is enabled, provide the username",
			Required: false,
			EnvVars:  []string{"BASIC_AUTH_USERNAME"},
		},
		&cli.StringFlag{
			Name:     "basic-auth-password",
			Usage:    "If Basic Auth is enabled, provide the password",
			Required: false,
			EnvVars:  []string{"BASIC_AUTH_PASSWORD"},
		},
		&cli.BoolFlag{
			Name:     "enable-unknown-queue-items",
			Usage:    "Enable gathering unknown queue items in Queue metrics",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_UNKNOWN_QUEUE_ITEMS"},
		},
	}
	if whatarr == "sonarr" {
		flags = append(flags, &cli.BoolFlag{
			Name:     "enable-episode-quality-metrics",
			Usage:    "Enable gathering total episodes by qualities",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_EPISODE_QUALITY_METRICS"},
		})
	}
	if whatarr == "lidarr" {
		flags = append(flags, &cli.BoolFlag{
			Name:     "enable-song-quality-metrics",
			Usage:    "Enable gathering total songs by quality.",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_SONG_QUALITY_METRICS"},
		})
		flags = append(flags, &cli.BoolFlag{
			Name:     "enable-monitored-albums-metrics",
			Usage:    "Enable gathering monitored albums.",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_MONITORED_ALBUMS_METRICS"},
		})
	}
	return flags
}
