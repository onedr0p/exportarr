package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

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
	app.Usage = "A Prometheus Exporter for Sonarr, Radarr and Lidarr"
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
			Usage:    "Set the default Log Level",
			Value:    "INFO",
			Required: false,
			EnvVars:  []string{"LOG_LEVEL"},
		},
		&cli.IntFlag{
			Name:     "listen-port",
			Usage:    "Port the exporter will listen on",
			Value:    9707,
			Required: false,
			EnvVars:  []string{"LISTEN_PORT"},
		},
		&cli.StringFlag{
			Name:     "listen-ip",
			Usage:    "IP the exporter will listen on",
			Value:    "0.0.0.0",
			Required: false,
			EnvVars:  []string{"LISTEN_IP"},
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "url",
					Value:    "http://127.0.0.1:7878",
					Usage:    "Full URL to Radarr",
					Required: false,
					EnvVars:  []string{"RADARR_URL"},
				},
				&cli.StringFlag{
					Name:     "api-key",
					Usage:    "Radarr's API Key",
					Required: true,
					EnvVars:  []string{"RADARR_APIKEY"},
				},
			},
			Action: radarr,
			Before: func(c *cli.Context) error {
				if !utils.IsValidUrl(c.String("url")) {
					return cli.Exit(fmt.Sprintf("%s is not a valid URL", c.String("url")), 10)
				}
				if !utils.IsValidApikey(c.String("api-key")) {
					return cli.Exit(fmt.Sprintf("%s is not a valid API Key", c.String("api-key")), 11)
				}
				return nil
			},
		},
		{
			Name:        "sonarr",
			Aliases:     []string{"s"},
			Usage:       "Use the exporter for Sonarr",
			Description: strings.Title("Sonarr Exporter"),
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "url",
					Value:    "http://127.0.0.1:8989",
					Usage:    "Full URL to Sonarr",
					Required: false,
					EnvVars:  []string{"SONARR_URL"},
				},
				&cli.StringFlag{
					Name:     "api-key",
					Usage:    "Sonarr's API Key",
					Required: true,
					EnvVars:  []string{"SONARR_APIKEY"},
				},
				&cli.BoolFlag{
					Name:     "disable-episode-quality-metrics",
					Usage:    "Disable getting Episode qualities",
					Value:    true,
					Required: false,
					EnvVars:  []string{"SONARR_DISABLE_EPISODE_QUALITY_METRICS"},
				},
			},
			Action: sonarr,
			Before: func(c *cli.Context) error {
				if !utils.IsValidUrl(c.String("url")) {
					return cli.Exit(fmt.Sprintf("%s is not a valid URL", c.String("url")), 10)
				}
				if !utils.IsValidApikey(c.String("api-key")) {
					return cli.Exit(fmt.Sprintf("%s is not a valid API Key", c.String("api-key")), 11)
				}
				return nil
			},
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

func serveHttp(c *cli.Context, r *prometheus.Registry) error {
	// Set up the handlers
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})
	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/healthz", handlers.HealthzHandler)
	http.Handle("/metrics", handler)
	// Serve up the metrics
	log.Infof("Listening on %s:%d", c.String("listen-ip"), c.Int("listen-port"))
	httpErr := http.ListenAndServe(
		fmt.Sprintf("%s:%d", c.String("listen-ip"), c.Int("listen-port")),
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
