package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/handlers"
)

var GRACEFUL_TIMEOUT = 5 * time.Second

var (
	conf = &config.Config{}

	rootCmd = &cobra.Command{
		Use:   "exportarr",
		Short: "exportarr is a AIO Prometheus exporter for *arr applications",
		Long: `exportarr is a Prometheus exporter for *arr applications.
It can export metrics from Radarr, Sonarr, Lidarr, Readarr, and Prowlarr.
More information available at the Github Repo (https://github.com/onedr0p/exportarr)`,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(validateFlags, initConfig)

	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringP("config", "c", "", "*arr config.xml file for parsing authentication information")
	rootCmd.PersistentFlags().StringP("url", "u", "", "URL to *arr instance")
	rootCmd.PersistentFlags().StringP("api-key", "k", "", "API Key for *arr instance")
	rootCmd.PersistentFlags().StringP("api-key-file", "f", "", "File containing API Key for *arr instance")
	rootCmd.PersistentFlags().Int("port", 0, "Port to listen on")
	rootCmd.PersistentFlags().StringP("interface", "i", "", "IP address to listen on")
	rootCmd.PersistentFlags().Bool("disable-ssl-verify", false, "Disable SSL verification")
	rootCmd.PersistentFlags().String("auth-username", "", "Username for basic auth")
	rootCmd.PersistentFlags().String("auth-password", "", "Password for basic auth")
	rootCmd.PersistentFlags().Bool("form-auth", false, "Use form based authentication")
	rootCmd.PersistentFlags().Bool("enable-unknown-queue-items", false, "Enable unknown queue items")
	rootCmd.PersistentFlags().Bool("enable-additional-metric", false, "Enable additional metric")
}

// validateFlags performs logical validation of flags, content is validated in the config package
func validateFlags() {
	flags := rootCmd.PersistentFlags()
	apiKey, _ := flags.GetString("api-key")
	apiKeyFile, _ := flags.GetString("api-key-file")
	apiKeySet := apiKey != "" || apiKeyFile != ""
	url, _ := flags.GetString("url")
	xmlConfig, _ := flags.GetString("config")

	err := validation.Errors{
		"api-key": validation.Validate(apiKey,
			validation.When(apiKeyFile != "", validation.Empty.Error("api-key and api-key-file are mutually exclusive")),
		),
		"api-key-file": validation.Validate(apiKeyFile,
			validation.When(apiKey != "", validation.Empty.Error("api-key and api-key-file are mutually exclusive")),
		),
		"config": validation.Validate(xmlConfig,
			validation.When(url == "" || !apiKeySet, validation.Required.Error("url & api-key must be set, or config must be set")),
			validation.When(url != "" && apiKeySet, validation.Empty.Error("only two of url, api-key/api-key-file. and config can be set")),
			validation.When(apiKey == "" && apiKeyFile == "", validation.Required.Error("one of api-key, api-key-file, or config is required")),
		),
	}
	if err.Filter() != nil {
		for _, e := range err {
			log.Fatal(e)
		}
	}
}

func initConfig() {
	var err error
	conf, err = config.LoadConfig(rootCmd.PersistentFlags())
	if err != nil {
		log.Fatal(err)
	}

	if err := conf.Validate(); err != nil {
		log.Fatal(err)
	}
	level, err := log.ParseLevel(conf.LogLevel)
	if err != nil {
		log.Errorf("Invalid log level %s, using default level: info", conf.LogLevel)
		level = log.InfoLevel
	}
	log.SetLevel(level)
}

type registerFunc func(registry *prometheus.Registry)

func serveHttp(fn registerFunc) {
	var srv http.Server

	idleConnsClosed := make(chan struct{})
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		signal.Notify(sigchan, syscall.SIGTERM)
		sig := <-sigchan
		log.Infof("Received signal %s, shutting down", sig)

		ctx, cancel := context.WithTimeout(context.Background(), GRACEFUL_TIMEOUT)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed: %v", err)
		}
		close(idleConnsClosed)
	}()

	registry := prometheus.NewRegistry()
	fn(registry)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/healthz", handlers.HealthzHandler)
	http.Handle("/metrics", handler)

	log.Infof("Listening on %s:%d", conf.Interface, conf.Port)
	srv.Addr = fmt.Sprintf("%s:%d", conf.Interface, conf.Port)
	srv.Handler = logRequest(http.DefaultServeMux)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Failed to Start HTTP Server: %v", err)
	}
	<-idleConnsClosed
}

// Log internal request to stdout
func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
