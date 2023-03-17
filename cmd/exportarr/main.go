package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	lidarrCollector "github.com/onedr0p/exportarr/internal/collector/lidarr"
	prowlarrCollector "github.com/onedr0p/exportarr/internal/collector/prowlarr"
	radarrCollector "github.com/onedr0p/exportarr/internal/collector/radarr"
	readarrCollector "github.com/onedr0p/exportarr/internal/collector/readarr"
	sharedCollector "github.com/onedr0p/exportarr/internal/collector/shared"
	sonarrCollector "github.com/onedr0p/exportarr/internal/collector/sonarr"
	"github.com/onedr0p/exportarr/internal/model"

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
		&cli.Author{
			Name:  "kinduff",
			Email: "313nyk550@relay.firefox.com",
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
	app.Before = func(config *cli.Context) error {
		switch strings.ToUpper(config.String("log-level")) {
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
			Name:        "lidarr",
			Aliases:     []string{"l"},
			Usage:       "Prometheus Exporter for Lidarr",
			Description: strings.Title("Lidarr Exporter"),
			Flags:       flags("lidarr"),
			Action:      lidarr,
			Before:      validation,
		},
		{
			Name:        "radarr",
			Aliases:     []string{"r"},
			Usage:       "Prometheus Exporter for Radarr",
			Description: strings.Title("Radarr Exporter"),
			Flags:       flags("radarr"),
			Action:      radarr,
			Before:      validation,
		},
		{
			Name:        "sonarr",
			Aliases:     []string{"s"},
			Usage:       "Prometheus Exporter for Sonarr",
			Description: strings.Title("Sonarr Exporter"),
			Flags:       flags("sonarr"),
			Action:      sonarr,
			Before:      validation,
		},
		{
			Name:        "prowlarr",
			Aliases:     []string{"p"},
			Usage:       "Prometheus Exporter for Prowlarr",
			Description: strings.Title("Prowlarr Exporter"),
			Flags:       flags("prowlarr"),
			Action:      prowlarr,
			Before:      validation,
		},
		{
			Name:        "readarr",
			Aliases:     []string{"b"}, // b for book
			Usage:       "Prometheus Exporter for Readarr",
			Description: strings.Title("Readarr Exporter"),
			Flags:       flags("readarr"),
			Action:      readarr,
			Before:      validation,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func lidarr(config *cli.Context) (err error) {
	registry := prometheus.NewRegistry()

	var configFile *model.Config
	if config.String("config") != "" {
		configFile, _ = utils.GetArrConfigFromFile(config.String("config"))
	} else {
		configFile = model.NewConfig()
	}
	configFile.ApiVersion = "v1"

	registry.MustRegister(
		lidarrCollector.NewLidarrCollector(config, configFile),
		sharedCollector.NewQueueCollector(config, configFile),
		sharedCollector.NewHistoryCollector(config, configFile),
		sharedCollector.NewRootFolderCollector(config, configFile),
		sharedCollector.NewSystemStatusCollector(config, configFile),
		sharedCollector.NewSystemHealthCollector(config, configFile),
	)
	return serveHttp(config, registry)
}

func radarr(config *cli.Context) (err error) {
	registry := prometheus.NewRegistry()

	var configFile *model.Config
	if config.String("config") != "" {
		configFile, _ = utils.GetArrConfigFromFile(config.String("config"))
	} else {
		configFile = model.NewConfig()
	}

	registry.MustRegister(
		radarrCollector.NewRadarrCollector(config, configFile),
		sharedCollector.NewQueueCollector(config, configFile),
		sharedCollector.NewHistoryCollector(config, configFile),
		sharedCollector.NewRootFolderCollector(config, configFile),
		sharedCollector.NewSystemStatusCollector(config, configFile),
		sharedCollector.NewSystemHealthCollector(config, configFile),
	)
	return serveHttp(config, registry)
}

func sonarr(config *cli.Context) (err error) {
	registry := prometheus.NewRegistry()

	var configFile *model.Config
	if config.String("config") != "" {
		configFile, _ = utils.GetArrConfigFromFile(config.String("config"))
	} else {
		configFile = model.NewConfig()
	}

	registry.MustRegister(
		sonarrCollector.NewSonarrCollector(config, configFile),
		sharedCollector.NewQueueCollector(config, configFile),
		sharedCollector.NewHistoryCollector(config, configFile),
		sharedCollector.NewRootFolderCollector(config, configFile),
		sharedCollector.NewSystemStatusCollector(config, configFile),
		sharedCollector.NewSystemHealthCollector(config, configFile),
	)
	return serveHttp(config, registry)
}

func prowlarr(config *cli.Context) (err error) {
	registry := prometheus.NewRegistry()

	var configFile *model.Config
	if config.String("config") != "" {
		configFile, _ = utils.GetArrConfigFromFile(config.String("config"))
	} else {
		configFile = model.NewConfig()
	}
	configFile.ApiVersion = "v1"

	registry.MustRegister(
		prowlarrCollector.NewProwlarrCollector(config, configFile),
		sharedCollector.NewHistoryCollector(config, configFile),
		sharedCollector.NewSystemStatusCollector(config, configFile),
		sharedCollector.NewSystemHealthCollector(config, configFile),
	)
	return serveHttp(config, registry)
}

func readarr(config *cli.Context) (err error) {
	registry := prometheus.NewRegistry()

	var configFile *model.Config
	if config.String("config") != "" {
		configFile, _ = utils.GetArrConfigFromFile(config.String("config"))
	} else {
		configFile = model.NewConfig()
	}
	configFile.ApiVersion = "v1"

	registry.MustRegister(
		readarrCollector.NewReadarrCollector(config, configFile),
		sharedCollector.NewQueueCollector(config, configFile),
		sharedCollector.NewHistoryCollector(config, configFile),
		sharedCollector.NewRootFolderCollector(config, configFile),
		sharedCollector.NewSystemStatusCollector(config, configFile),
		sharedCollector.NewSystemHealthCollector(config, configFile),
	)
	return serveHttp(config, registry)
}

func serveHttp(config *cli.Context, registry *prometheus.Registry) error {
	// Set up the handlers
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/healthz", handlers.HealthzHandler)
	http.Handle("/metrics", handler)
	// Serve up the metrics
	log.Infof("Listening on %s:%d", config.String("interface"), config.Int("port"))
	httpErr := http.ListenAndServe(
		fmt.Sprintf("%s:%d", config.String("interface"), config.Int("port")),
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
func validation(config *cli.Context) error {
	// Data validations
	if config.String("url") != "" && !utils.IsValidUrl(config.String("url")) {
		return cli.Exit(fmt.Sprintf("%s is not a valid URL", config.String("url")), 1)
	}
	if config.String("api-key") != "" && !utils.IsValidApikey(config.String("api-key")) {
		return cli.Exit(fmt.Sprintf("%s is not a valid API Key", config.String("api-key")), 1)
	}
	if config.String("api-key-file") != "" {
		data, err := os.ReadFile(config.String("api-key-file"))
		if err != nil {
			return cli.Exit(fmt.Sprintf("unable to read API Key file %s", config.String("api-key-file")), 1)
		}
		if !utils.IsValidApikey(string(data)) {
			return cli.Exit(fmt.Sprintf("%s is not a valid API Key", string(data)), 1)
		}
	}
	if config.String("config") != "" &&
		!utils.IsFileThere(config.String("config")) {
		return cli.Exit(fmt.Sprintf("%s config file does not exist", config.String("config")), 1)
	}

	// Logical validations
	if config.String("api-key") != "" && config.String("api-key-file") != "" {
		return cli.Exit("either api-key or api-key-file can be set, not both of them", 1)
	}
	apiKeyIsSet := config.String("api-key") != "" || config.String("api-key-file") != ""
	if config.String("url") != "" && apiKeyIsSet && config.String("config") != "" {
		return cli.Exit("url and api-key or config must be set, not all of them", 1)
	}
	if config.String("url") == "" && !apiKeyIsSet && config.String("config") == "" {
		return cli.Exit("url and api-key or config must be set, not none of them", 1)
	}
	if config.Bool("form-auth") && (config.String("auth-username") == "" || config.String("auth-password") == "") {
		return cli.Exit("username and password must be set if form-auth is set", 1)
	}
	return nil
}

// Flags used for all services
func flags(arr string) []cli.Flag {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:     "url",
			Aliases:  []string{"u"},
			Usage:    fmt.Sprintf("%s's full URL", arr),
			Required: true,
			EnvVars:  []string{"URL"},
		},
		&cli.StringFlag{
			Name:     "api-key",
			Aliases:  []string{"a"},
			Usage:    fmt.Sprintf("%s's API Key", arr),
			Required: false,
			EnvVars:  []string{"APIKEY"},
			FilePath: "/etc/exportarr/apikey",
		},
		&cli.StringFlag{
			Name:     "api-key-file",
			Usage:    fmt.Sprintf("%s's API Key file location", arr),
			Required: false,
			EnvVars:  []string{"APIKEY_FILE"},
		},
		&cli.StringFlag{
			Name:     "config",
			Aliases:  []string{"c"},
			Usage:    fmt.Sprintf("Path to %s's config.xml", arr),
			Required: false,
			EnvVars:  []string{"CONFIG"},
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
		&cli.StringFlag{
			Name:     "auth-username",
			Aliases:  []string{"basic-auth-username"},
			Usage:    "Provide the username for basic or form auth",
			Required: false,
			EnvVars:  []string{"AUTH_USERNAME", "BASIC_AUTH_USERNAME"},
		},
		&cli.StringFlag{
			Name:     "auth-password",
			Aliases:  []string{"basic-auth-password"},
			Usage:    "Provide the password for basic or form auth",
			Required: false,
			EnvVars:  []string{"AUTH_PASSWORD", "BASIC_AUTH_PASSWORD"},
		},
		&cli.BoolFlag{
			Name:     "form-auth",
			Usage:    "Use form authentication rather than basic auth",
			Value:    false,
			Required: false,
			EnvVars:  []string{"FORM_AUTH"},
		},
		&cli.BoolFlag{
			Name:     "enable-unknown-queue-items",
			Usage:    "Enable gathering of unknown queue items in Queue metrics",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_UNKNOWN_QUEUE_ITEMS"},
		},
		&cli.BoolFlag{
			Name:     "enable-additional-metrics",
			Usage:    "Enable gathering of additional metrics (will slow down metrics gathering)",
			Value:    false,
			Required: false,
			EnvVars:  []string{"ENABLE_ADDITIONAL_METRICS"},
		},
	}
	return flags
}
