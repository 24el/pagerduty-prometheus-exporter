package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/24el/pagerduty-prometheus-exporter/cmd/pagerduty-prometheus-exporter/cmd/httphandler"
	"github.com/24el/pagerduty-prometheus-exporter/cmd/pagerduty-prometheus-exporter/cmd/middleware"
	"github.com/24el/pagerduty-prometheus-exporter/internal/collector"
	"github.com/24el/pagerduty-prometheus-exporter/internal/collector/webhook"
	"github.com/24el/pagerduty-prometheus-exporter/internal/pagerduty"
)

type srvShutdowner func(context.Context) error

type options struct {
	MetricsSrvPort int
	WebhookSrvPort int

	IncidentWebhookSignatureSecret string `envconfig:"incident_webhook_signature_secret"`
	IncidentWebhookPath            string

	MetricsPrefix               string
	AnalyticsScrapeInterval     time.Duration
	AnalyticsReportPeriods      []time.Duration
	AnalyticsServiceMetricNames []string
	UsersScrapeInterval         time.Duration

	DTFormat string

	PagerdutyAuthToken string `envconfig:"pagerduty_auth_token"`
	Debug              bool
}

func NewPagerdutyPrometheusExporterCommand() *cobra.Command {
	var o options

	cmd := &cobra.Command{
		Use:   "pagerduty-prometheus-exporter",
		Short: "Exports pagerduty to prometheus",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := createLogger(o.Debug)
			if err != nil {
				return err
			}

			err = envconfig.Process("", &o)
			if err != nil {
				return err
			}

			return run(logger, &o)
		},
	}

	flags := cmd.Flags()

	flags.IntVar(&o.MetricsSrvPort, "metrics-srv-port", 9100, "metrics server port")
	flags.IntVar(&o.WebhookSrvPort, "webhook-srv-port", 8080, "webhook server port")
	flags.StringVar(&o.IncidentWebhookSignatureSecret, "incident-webhook-signature-secret", "", "incident webhook signature secret")
	flags.StringVar(&o.IncidentWebhookPath, "incident-webhook-path", "/v1/incidents", "incident webhook path")
	flags.StringVar(&o.MetricsPrefix, "metrics-prefix", "", "metrics prefix")
	flags.DurationVar(&o.AnalyticsScrapeInterval, "analytics-scrape-interval", time.Minute, "scrape service analytic metric interval")
	flags.StringSliceVar(
		&o.AnalyticsServiceMetricNames,
		"analytics-service-metric-names",
		[]string{
			string(pagerduty.ReportMetricTotalEscalationsCount),
			string(pagerduty.ReportMetricIncidentCount),
			string(pagerduty.ReportMetricMeanSecondsToResolve),
			string(pagerduty.ReportMetricMeanSecondsToFirstAck),
			string(pagerduty.ReportMetricIncidentUpTimePct),
		},
		"scrape service analytic metric names",
	)
	flags.DurationSliceVar(
		&o.AnalyticsReportPeriods,
		"analytics-report-periods",
		[]time.Duration{time.Hour * 24 * 90},
		"scrape service analytic metric periods",
	)
	flags.DurationVar(&o.UsersScrapeInterval, "users-scrape-interval", 5*time.Minute, "scrape users interval")
	flags.StringVar(&o.DTFormat, "dt-format", time.RFC3339, "dt format")
	flags.StringVar(&o.PagerdutyAuthToken, "pagerduty-auth-token", "", "pagerduty auth token")
	flags.BoolVar(&o.Debug, "debug", false, "debug")

	return cmd
}

func run(logger *zap.Logger, opts *options) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	eg, gCtx := errgroup.WithContext(ctx)

	metricsSrv := createMetricsServer(opts)

	registerer := prometheus.WrapRegistererWithPrefix(opts.MetricsPrefix, prometheus.DefaultRegisterer)

	collectors, err := resolvePagerdutyMetricCollectors(logger, registerer, opts)
	if err != nil {
		return errors.Wrap(err, "resolve metric collectors")
	}

	srvShutdowners := []srvShutdowner{metricsSrv.Shutdown}

	eg.Go(func() error {
		logger.Info("Starting metrics server", zap.String("addr", metricsSrv.Addr))

		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errors.Wrap(err, "listen and serve metrics server")
		}

		return nil
	})

	if opts.WebhookSrvPort != 0 {
		webhookSrv := createWebhookServer(logger, registerer, opts)

		srvShutdowners = append(srvShutdowners, webhookSrv.Shutdown)

		eg.Go(func() error {
			logger.Info("Starting webhook server", zap.String("addr", webhookSrv.Addr))

			if err := webhookSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return errors.Wrap(err, "listen and serve webhook server")
			}

			return nil
		})
	}

	for i := range collectors {
		cl := collectors[i]
		eg.Go(func() error {
			return cl.Collect(gCtx)
		})
	}

	go func() {
		<-gCtx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		for _, shutdowner := range srvShutdowners {
			if err := shutdowner(shutdownCtx); err != nil {
				logger.Error("server shutdown failed", zap.Error(err))
			}
		}
	}()

	return eg.Wait()
}

func createMetricsServer(opts *options) *http.Server {
	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", opts.MetricsSrvPort),
		Handler: r,
	}
}

func createWebhookServer(logger *zap.Logger, registerer prometheus.Registerer, opts *options) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", opts.WebhookSrvPort),
		Handler: setupWebhookHTTPHandler(logger, registerer, opts),
	}
}

func setupWebhookHTTPHandler(logger *zap.Logger, registerer prometheus.Registerer, opts *options) http.Handler {
	serveMux := mux.NewRouter()
	serveMux.Use(
		middleware.HTTPPrometheusMetrics(registerer),
	)

	incidentListener := webhook.NewIncidentMetricsListener(opts.DTFormat, registerer)

	webhookHandler := httphandler.NewWebhookHandler(
		logger,
		incidentListener,
		[]byte(opts.IncidentWebhookSignatureSecret),
	)
	webhookHandler.InstallRoutes(serveMux, opts.IncidentWebhookPath)

	recovery := gorillahandlers.RecoveryHandler(gorillahandlers.PrintRecoveryStack(true))

	return recovery(serveMux)
}

func resolvePagerdutyMetricCollectors(
	logger *zap.Logger,
	registerer prometheus.Registerer,
	opts *options,
) ([]collector.Interface, error) {
	serviceAnalyticsCollectors := make([]collector.Interface, len(opts.AnalyticsReportPeriods))

	collectProcessMetrics := collector.RegisterCollectProcessMetrics(registerer)

	serviceMetricNames, err := resolveReportMetricNames(opts)
	if err != nil {
		return nil, err
	}

	serviceAnalyticMetrics := collector.RegisterServiceAnalyticMetricsFromNames(registerer, serviceMetricNames)

	pdExtendedClient := pagerduty.NewExtendedClient(opts.PagerdutyAuthToken)

	for i := range opts.AnalyticsReportPeriods {
		serviceAnalyticsCollectors[i] = collector.NewGracefulCollectorWithMetrics(
			logger,
			collectProcessMetrics,
			"service_analytics",
			collector.NewServiceAnalyticsCollector(
				logger,
				pdExtendedClient,
				serviceAnalyticMetrics,
				serviceMetricNames,
				opts.AnalyticsReportPeriods[i],
			),
		)
	}

	serviceAnalyticsCollector := collector.NewPeriodicCollector(
		opts.AnalyticsScrapeInterval,
		serviceAnalyticsCollectors...,
	)

	usersCollector := collector.NewPeriodicCollector(
		opts.UsersScrapeInterval,
		collector.NewGracefulCollectorWithMetrics(
			logger,
			collectProcessMetrics,
			"users",
			collector.NewUsersCollector(pdExtendedClient, registerer),
		),
	)

	return []collector.Interface{serviceAnalyticsCollector, usersCollector}, nil
}

func resolveReportMetricNames(opts *options) ([]pagerduty.ReportMetricName, error) {
	rm := make([]pagerduty.ReportMetricName, len(opts.AnalyticsServiceMetricNames))

	for i := range opts.AnalyticsServiceMetricNames {
		mn, err := pagerduty.GetReportMetricName(opts.AnalyticsServiceMetricNames[i])
		if err != nil {
			return nil, err
		}

		rm[i] = mn
	}

	return rm, nil
}

func createLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
