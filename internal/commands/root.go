package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	"github.com/onedr0p/exportarr/internal/config"
	"github.com/onedr0p/exportarr/internal/handlers"
)

const gracefulTimeout = 5 * time.Second

var (
	conf    = &config.Config{}
	appInfo = &AppInfo{}
	rootCmd = &cobra.Command{
		Use:   "exportarr",
		Short: "exportarr is a AIO Prometheus exporter for *arr applications",
		Long: `exportarr is a Prometheus exporter for *arr applications.
It can export metrics from Radarr, Sonarr, Lidarr, Bazarr, Prowlarr and SABnzbd.
More information available at the Github Repo (https://github.com/onedr0p/exportarr)`,
		// Load + validate config and install the logger before any subcommand
		// runs, returning errors instead of exiting so cobra can report them
		// and defers still run. Help and completion skip config entirely.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			switch cmd.Name() {
			case "help", "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
				return nil
			}
			var err error
			conf, err = config.LoadConfig(cmd.Root().PersistentFlags())
			if err != nil {
				return err
			}
			conf.App = cmd.Name()
			if err := conf.Validate(); err != nil {
				return err
			}
			initLogger()
			return nil
		},
	}
)

// AppInfo carries build metadata stamped into the binary at release time.
type AppInfo struct {
	Name      string
	Version   string
	BuildTime string
	Revision  string
}

// Execute runs the exportarr root command.
func Execute(a AppInfo) error {
	appInfo = &a
	return rootCmd.Execute()
}

func init() {
	config.RegisterConfigFlags(rootCmd.PersistentFlags())
}

func initLogger() {
	// Install the handler at info first so config failures are loggable, then
	// lower/raise the level once the configured value parses.
	lvl := new(slog.LevelVar)

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	if conf.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}
	slog.SetDefault(slog.New(handler))

	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(conf.LogLevel)); err != nil {
		slog.Error("Invalid log level, using default level: info", "log-level", conf.LogLevel)
		parsed = slog.LevelInfo
	}
	lvl.Set(parsed)

	slog.Info(
		fmt.Sprintf("Starting %s", appInfo.Name),
		"app_name", appInfo.Name,
		"version", appInfo.Version,
		"buildTime", appInfo.BuildTime,
		"revision", appInfo.Revision,
	)
}

// promhttpLogger routes promhttp's internal gather errors to slog.
type promhttpLogger struct{}

// Println implements promhttp.Logger.
func (promhttpLogger) Println(v ...any) {
	slog.Error(fmt.Sprintln(v...))
}

type registerFunc func(registry prometheus.Registerer)

func serveHTTP(fn registerFunc) error {
	srv := http.Server{
		// Bound header reads so a stalled client cannot pin connections open.
		ReadHeaderTimeout: 10 * time.Second,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, os.Interrupt)
		signal.Notify(sigchan, syscall.SIGTERM)
		sig := <-sigchan
		slog.Info("Shutting down due to signal", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), gracefulTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("Server shutdown failed", "error", err)
			os.Exit(1)
		}
		close(idleConnsClosed)
	}()

	registry := prometheus.NewRegistry()
	registerAppInfoMetric(registry)
	// The exporter's own runtime health: go_* and process_* metrics make its
	// CPU, memory, and GC behavior visible to the operator.
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	fn(registry)

	// Serve partial metrics when a collector fails rather than failing the
	// whole scrape; collectors surface failures via their *_collector_error
	// gauges. Scrape bookkeeping wraps only /metrics so health probes don't
	// pollute it.
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
		ErrorLog:      promhttpLogger{},
	})
	mux := http.NewServeMux()
	mux.Handle("/metrics", handlers.MetricsHandler(conf, registry, metricsHandler))
	mux.HandleFunc("/", handlers.IndexHandler)
	mux.HandleFunc("/healthz", handlers.HealthzHandler)

	slog.Info("Starting HTTP Server",
		"interface", conf.Interface,
		"port", conf.Port)
	srv.Addr = fmt.Sprintf("%s:%d", conf.Interface, conf.Port)

	wrappedMux := handlers.RecoveryHandler(mux)
	wrappedMux = handlers.LogHandler(wrappedMux)

	srv.Handler = wrappedMux

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	<-idleConnsClosed
	return nil
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
