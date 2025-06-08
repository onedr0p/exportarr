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

	"github.com/shamelin/exportarr/internal/config"
	"github.com/shamelin/exportarr/internal/handlers"
)

var GRACEFUL_TIMEOUT = 5 * time.Second

var (
	conf    = &config.Config{}
	appInfo = &AppInfo{}
	rootCmd = &cobra.Command{
		Use:   "exportarr",
		Short: "exportarr is a AIO Prometheus exporter for *arr applications",
		Long: `exportarr is a Prometheus exporter for *arr applications.
It can export metrics from Radarr, Sonarr, Lidarr, Readarr, Bazarr and Prowlarr.
More information available at the Github Repo (https://github.com/shamelin/exportarr)`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			conf.App = cmd.Name()
		},
	}
)

type AppInfo struct {
	Name      string
	Version   string
	BuildTime string
	Revision  string
}

func Execute(a AppInfo) error {
	appInfo = &a
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)
	cobra.OnFinalize(finalizeLogger)

	config.RegisterConfigFlags(rootCmd.PersistentFlags())
}

func initConfig() {
	var err error
	conf, err = config.LoadConfig(rootCmd.PersistentFlags())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		if err := rootCmd.Usage(); err != nil {
			panic(err)
		}
		os.Exit(1)
	}

	if err := conf.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if err := rootCmd.Usage(); err != nil {
			panic(err)
		}
		os.Exit(1)
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

	zap.S().Infow(
		fmt.Sprintf("Starting %s", appInfo.Name),
		"app_name", appInfo.Name,
		"version", appInfo.Version,
		"buildTime", appInfo.BuildTime,
		"revision", appInfo.Revision,
	)
}

func finalizeLogger() {
	// Flushes buffered log messages
	zap.S().Sync() //nolint:errcheck
}

type registerFunc func(registry prometheus.Registerer)

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
	registerAppInfoMetric(registry)
	fn(registry)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/healthz", handlers.HealthzHandler)

	zap.S().Infow("Starting HTTP Server",
		"interface", conf.Interface,
		"port", conf.Port)
	srv.Addr = fmt.Sprintf("%s:%d", conf.Interface, conf.Port)

	wrappedMux := handlers.RecoveryHandler(mux)
	wrappedMux = handlers.MetricsHandler(conf, registry, wrappedMux)
	wrappedMux = handlers.LogHandler(wrappedMux)

	srv.Handler = wrappedMux

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		zap.S().Fatalw("Failed to Start HTTP Server",
			"error", err)
	}
	<-idleConnsClosed
}

func registerAppInfoMetric(registry prometheus.Registerer) {
	registry.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: appInfo.Name,
			Name:      "app_info",
			Help:      "A metric with a constant '1' value labeled by app name, version, build time, and revision.",
			ConstLabels: prometheus.Labels{
				"app_name":   appInfo.Name,
				"version":    appInfo.Version,
				"build_time": appInfo.BuildTime,
				"revision":   appInfo.Revision,
			},
		},
		func() float64 { return 1 },
	))
}
