package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/24el/pagerduty-prometheus-exporter/internal/pagerduty"
)

const UTCTimeZone = "Etc/UTC"

var metricNameReplacer = strings.NewReplacer(":", "_", ".", "_")

type ServiceAnalyticMetrics map[pagerduty.ReportMetricName]*prometheus.GaugeVec

func (m *ServiceAnalyticMetrics) prepareMetricName(metricName string) string {
	return fmt.Sprintf("pagerduty_service_%s", metricNameReplacer.Replace(metricName))
}

func RegisterServiceAnalyticMetricsFromNames(
	registerer prometheus.Registerer,
	metricNames []pagerduty.ReportMetricName,
) ServiceAnalyticMetrics {
	gaugeMetrics := make(ServiceAnalyticMetrics, len(metricNames))

	for _, mn := range metricNames {
		gaugeMetrics[mn] = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: gaugeMetrics.prepareMetricName(string(mn)),
			},
			[]string{"service_id", "service_name", "report_interval"},
		)

		registerer.MustRegister(gaugeMetrics[mn])
	}

	return gaugeMetrics
}

type ServiceAnalyticsCollector struct {
	logger      *zap.Logger
	client      pagerduty.Client
	metricNames []pagerduty.ReportMetricName
	interval    time.Duration

	metrics map[pagerduty.ReportMetricName]*prometheus.GaugeVec
}

func NewServiceAnalyticsCollector(
	logger *zap.Logger,
	client pagerduty.Client,
	serviceAnalyticMetrics ServiceAnalyticMetrics,
	metricNames []pagerduty.ReportMetricName,
	interval time.Duration,
) *ServiceAnalyticsCollector {
	return &ServiceAnalyticsCollector{
		logger:      logger,
		metricNames: metricNames,
		client:      client,
		interval:    interval,
		metrics:     serviceAnalyticMetrics,
	}
}

func (c *ServiceAnalyticsCollector) Collect(ctx context.Context) error {
	t := time.Now()

	report, err := c.client.QueryMetricReport(ctx, pagerduty.ServiceMetricReportParams{
		TimeZone: UTCTimeZone,
		Filters: pagerduty.ServiceMetricReportFilters{
			CreatedAtStart: pagerduty.ReportTime(t.Add(-c.interval)),
			CreatedAtEnd:   pagerduty.ReportTime(t),
		},
	})
	if err != nil {
		return errors.Wrap(err, "query metric report")
	}

	for _, srvMetrics := range report.Data {
		labels := prometheus.Labels{
			"service_id":      srvMetrics.ServiceID,
			"service_name":    srvMetrics.ServiceName,
			"report_interval": c.interval.String(),
		}

		for _, metricName := range c.metricNames {
			metricVal, err := srvMetrics.GetMetricByName(metricName)
			if err != nil {
				c.logger.Error("Get metric by name error, skipping...", zap.String("metric_name", string(metricName)))
				continue
			}

			c.metrics[metricName].With(labels).Set(metricVal)
		}
	}

	return nil
}
