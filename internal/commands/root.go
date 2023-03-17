package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
	cobra.OnInitialize(initConfig, initLogger)
	cobra.OnFinalize(finalizeLogger)

	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().String("log-format", "console", "Log format (console, json)")
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

func initConfig() {
	var err error
	conf, err = config.LoadConfig(rootCmd.PersistentFlags())
	if err != nil {
		zap.S().Fatal(err)
	}

	if err := conf.Validate(); err != nil {
		zap.S().Fatal(err)
	}
}

func initLogger() {
	atom := zap.NewAtomicLevel()

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	if conf.LogFormat == "json" {
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	}

	logger := zap.New(zapcore.NewCore(encoder, zapcore.Lock(os.Stdout), atom))
	// Create a logger with a default level first to ensure config failures are loggable.
	atom.SetLevel(zapcore.InfoLevel)
	zap.ReplaceGlobals(logger)

	lvl, err := zapcore.ParseLevel(conf.LogLevel)
	if err != nil {
		zap.S().Errorf("Invalid log level %s, using default level: info", conf.LogLevel)
		lvl = zapcore.InfoLevel
	}
	atom.SetLevel(lvl)

	zap.S().Debug("Logger initialized")
}

func finalizeLogger() {
	// Flushes buffered log messages
	zap.S().Sync()
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
		zap.S().Infof(
			"Shutting down due to signal: %s", sig)

		ctx, cancel := context.WithTimeout(context.Background(), GRACEFUL_TIMEOUT)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			zap.S().Fatalw("Server shutdown failed",
				"error", err)
		}
		close(idleConnsClosed)
	}()

	registry := prometheus.NewRegistry()
	fn(registry)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	http.HandleFunc("/", handlers.IndexHandler)
	http.HandleFunc("/healthz", handlers.HealthzHandler)
	http.Handle("/metrics", handler)

	zap.S().Infow("Starting HTTP Server",
		"interface", conf.Interface,
		"port", conf.Port)
	srv.Addr = fmt.Sprintf("%s:%d", conf.Interface, conf.Port)
	srv.Handler = logRequest(http.DefaultServeMux)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		zap.S().Fatalw("Failed to Start HTTP Server",
			"error", err)
	}
	<-idleConnsClosed
}

// Log internal request to stdout
func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		zap.S().Debugw("Request Received",
			"remote_addr", r.RemoteAddr,
			"method", r.Method,
			"url", r.URL)
		handler.ServeHTTP(w, r)
	})
}
