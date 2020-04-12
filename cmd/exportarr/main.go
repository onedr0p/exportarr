package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/onedr0p/exportarr/internal/collector"
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
	app.EnableBashCompletion = true
	app.HideVersion = true
	app.Authors = []*cli.Author{
		&cli.Author{
			Name:  "onedr0p",
			Email: "onedr0p@users.noreply.github.com",
		},
		&cli.Author{
			Name:  "DirtyCajunRice",
			Email: "DirtyCajunRice@users.noreply.github.com",
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:        "radarr",
			Aliases:     []string{"r"},
			Description: strings.Title("Radarr Exporter"),
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:     "listen-port",
					Usage:    "Port the exporter will listen on",
					Value:    9707,
					Required: false,
					EnvVars:  []string{"RADARR_LISTEN_PORT"},
				},
				&cli.StringFlag{
					Name:     "listen-ip",
					Usage:    "IP the exporter will listen on",
					Value:    "0.0.0.0",
					Required: false,
					EnvVars:  []string{"RADARR_LISTEN_IP"},
				},
				// Move to top level arg?
				&cli.StringFlag{
					Name:     "log-level",
					Usage:    "Set the default Log Level",
					Value:    "INFO",
					Required: false,
					EnvVars:  []string{"RADARR_LOG_LEVEL"},
				},
				&cli.BoolFlag{
					Name:     "disable-ssl-verify",
					Usage:    "Disable SSL Verifications",
					Value:    false,
					Required: false,
					EnvVars:  []string{"RADARR_DISABLE_SSL_VERIFY"},
				},
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
				&cli.BoolFlag{
					Name:     "basic-auth-enabled",
					Usage:    "Enable Basic Auth",
					Value:    false,
					Required: false,
					EnvVars:  []string{"RADARR_BASIC_AUTH_ENABLED"},
				},
				&cli.StringFlag{
					Name:     "basic-auth-username",
					Usage:    "If Basic Auth is enabled, provide the username",
					Required: false,
					EnvVars:  []string{"RADARR_BASIC_AUTH_USERNAME"},
				},
				&cli.StringFlag{
					Name:     "basic-auth-password",
					Usage:    "If Basic Auth is enabled, provide the password",
					Required: false,
					EnvVars:  []string{"RADARR_BASIC_AUTH_PASSWORD"},
				},
			},
			Action: radarr,
		},
		{
			Name:        "sonarr",
			Aliases:     []string{"s"},
			Description: strings.Title("Sonarr Exporter"),
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:     "listen-port",
					Usage:    "Port the exporter will listen on",
					Value:    9707,
					Required: false,
					EnvVars:  []string{"SONARR_LISTEN_PORT"},
				},
				&cli.StringFlag{
					Name:     "listen-ip",
					Usage:    "IP the exporter will listen on",
					Value:    "0.0.0.0",
					Required: false,
					EnvVars:  []string{"SONARR_LISTEN_IP"},
				},
				// Move to top level arg?
				&cli.StringFlag{
					Name:     "log-level",
					Usage:    "Set the default Log Level",
					Value:    "INFO",
					Required: false,
					EnvVars:  []string{"SONARR_LOG_LEVEL"},
				},
				&cli.BoolFlag{
					Name:     "disable-ssl-verify",
					Usage:    "Disable SSL Verifications",
					Value:    false,
					Required: false,
					EnvVars:  []string{"SONARR_DISABLE_SSL_VERIFY"},
				},
				&cli.StringFlag{
					Name:     "url",
					Value:    "http://127.0.0.1:7878",
					Usage:    "Full URL to Radarr",
					Required: false,
					EnvVars:  []string{"SONARR_URL"},
				},
				&cli.StringFlag{
					Name:     "api-key",
					Usage:    "Radarr's API Key",
					Required: true,
					EnvVars:  []string{"SONARR_APIKEY"},
				},
				&cli.BoolFlag{
					Name:     "basic-auth-enabled",
					Usage:    "Enable Basic Auth",
					Value:    false,
					Required: false,
					EnvVars:  []string{"SONARR_BASIC_AUTH_ENABLED"},
				},
				&cli.StringFlag{
					Name:     "basic-auth-username",
					Usage:    "If Basic Auth is enabled, provide the username",
					Required: false,
					EnvVars:  []string{"SONARR_BASIC_AUTH_USERNAME"},
				},
				&cli.StringFlag{
					Name:     "basic-auth-password",
					Usage:    "If Basic Auth is enabled, provide the password",
					Required: false,
					EnvVars:  []string{"SONARR_BASIC_AUTH_PASSWORD"},
				},
			},
			Action: sonarr,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func radarr(c *cli.Context) (err error) {
	// Wire up Logging
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
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	// Validate Url
	if !utils.IsValidUrl(c.String("url")) {
		log.Fatalf("%s is not a proper URL", c.String("url"))
	}

	// Validate API Key
	if !utils.IsValidApikey(c.String("api-key")) {
		log.Fatalf("%s is not a proper API Key", c.String("api-key"))
	}

	// Register the Collectors
	r := prometheus.NewRegistry()
	r.MustRegister(
		collector.NewMovieCollector(c),
		collector.NewQueueCollector(c),
		collector.NewHistoryCollector(c),
		collector.NewRootFolderCollector(c),
		collector.NewSystemStatusCollector(c),
		collector.NewSystemHealthCollector(c),
	)

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

func sonarr(c *cli.Context) (err error) {
	// Wire up Logging
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
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	// Validate Url
	if !utils.IsValidUrl(c.String("url")) {
		log.Fatalf("%s is not a proper URL", c.String("url"))
	}

	// Validate API Key
	if !utils.IsValidApikey(c.String("api-key")) {
		log.Fatalf("%s is not a proper API Key", c.String("api-key"))
	}

	// Register the Collectors
	r := prometheus.NewRegistry()
	r.MustRegister(
		collector.NewMovieCollector(c),
		collector.NewQueueCollector(c),
		collector.NewHistoryCollector(c),
		collector.NewRootFolderCollector(c),
		collector.NewSystemStatusCollector(c),
		collector.NewSystemHealthCollector(c),
	)

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
