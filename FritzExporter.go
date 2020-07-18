package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/wbwue/FritzExporter/pkg/config"
	"github.com/wbwue/FritzExporter/pkg/scraper"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
)

var (
	// Version gets defined by the build system.
	Version = "0.0.0"
	// Revision gets defined by the built system
	Revision = ""
	// BuildDate defines the date this binary was built.
	BuildDate string
	// GoVersion running this binary.
	GoVersion = runtime.Version()
)

func main() {
	cfg := config.NewConfig()
	app := &cli.App{
		Name:    "FritzExporter",
		Version: fmt.Sprintf("%s (%s)", Version, Revision),
		Usage:   "FritzExporter",
	}
	cli.HelpFlag = &cli.BoolFlag{
		Name:    "help",
		Aliases: []string{"h"},
		Usage:   "Show help",
	}

	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v"},
		Usage:   "Prints the current version",
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "fritzbox-url",
			Usage:       "URL to connect to",
			EnvVars:     []string{"FRITZ_FRITZBOX_URL"},
			Destination: &cfg.FritzBoxURL,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "username",
			Usage:       "Username to login into Fritz!Box",
			EnvVars:     []string{"FRITZ_USERNAME"},
			Destination: &cfg.Username,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "log-level",
			Value:       "info",
			Usage:       "Only log messages with given severity",
			EnvVars:     []string{"FRITZ_LOG_LEVEL"},
			Destination: &cfg.LogLevel,
		},
		&cli.StringFlag{
			Name:        "password",
			Usage:       "Password to login into Fritz!Box",
			EnvVars:     []string{"FRITZ_PASSWORD"},
			Required:    true,
			Destination: &cfg.Password,
		},
		&cli.StringFlag{
			Name:        "exporter-address",
			Value:       "0.0.0.0:9200",
			Usage:       "Address to bind the metrics server",
			EnvVars:     []string{"FRITZ_EXPORTER_METRICS_ADDRESS"},
			Destination: &cfg.MetricsAddress,
		},
	}

	app.Action = func(c *cli.Context) error {
		return execute(cfg)
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func execute(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var g run.Group
	{
		logger := setupLogging(cfg)
		logger = log.With(logger, "component", "fritz_exporter")
		level.Info(logger).Log(
			"msg", "starting fritzbox exporter",
			"version", Version,
			"revision", Revision,
			"buildDate", BuildDate,
			"goVersion", GoVersion,
		)

		s := scraper.NewScraper(cfg, logger)
		g.Add(func() error {
			return s.Run(ctx)
		}, func(_ error) {
			level.Info(logger).Log("msg", "shutting down socket server")
		})
	}
	{
		logger := setupLogging(cfg)
		logger = log.With(logger, "component", "metrics")

		promauto.NewGaugeFunc(
			prometheus.GaugeOpts{
				Name: "fritz_exporter_build_info",
				Help: "A metric with a constant '1' value labeled by version, revision, build_date, and goversion.",
				ConstLabels: prometheus.Labels{
					"version":    Version,
					"revision":   Revision,
					"build_date": BuildDate,
					"goversion":  GoVersion,
				},
			},
			func() float64 { return 1 },
		)
		m := http.NewServeMux()
		m.Handle("/metrics", promhttp.Handler())
		s := http.Server{
			Addr:    cfg.MetricsAddress,
			Handler: m,
		}
		g.Add(func() error {
			level.Info(logger).Log("msg", "starting metrics server", "addr", cfg.MetricsAddress)
			return s.ListenAndServe()
		}, func(_ error) {
			level.Info(logger).Log("msg", "shutting down metric server")
			if err := s.Shutdown(context.Background()); err != nil {
				level.Error(logger).Log("msg", "error shutting down metrics server", "error", err)
			}
		})
	}

	{
		sig := make(chan os.Signal)
		g.Add(func() error {
			signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
			<-sig
			return nil
		}, func(err error) {
			cancel()
			close(sig)
		})
	}
	if err := g.Run(); err != nil {
		return err
	}
	return nil
}

func setupLogging(cfg *config.Config) log.Logger {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))

	switch strings.ToLower(cfg.LogLevel) {
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	default:
		logger = level.NewFilter(logger, level.AllowInfo())
	}
	logger = log.With(logger,
		"ts", log.DefaultTimestampUTC,
		"caller", log.DefaultCaller,
	)
	return logger
}
