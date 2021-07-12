package collector

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type CollectProcessMetrics struct {
	collectionLatencyHistogram *prometheus.HistogramVec
	collectionsCounter         *prometheus.CounterVec
	collectionErrorsCounter    *prometheus.CounterVec
}

func RegisterCollectProcessMetrics(registerer prometheus.Registerer) *CollectProcessMetrics {
	var (
		collectionLatencyHist = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "pagerduty_metrics_collector_latency",
			},
			[]string{"collector_name"},
		)
		collectionsCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pagerduty_metrics_collector_collections_count",
			},
			[]string{"collector_name"},
		)
		collectionErrorsCounter = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pagerduty_metrics_collector_errors_count",
			},
			[]string{"collector_name"},
		)
	)

	registerer.MustRegister(collectionsCounter, collectionErrorsCounter, collectionLatencyHist)

	return &CollectProcessMetrics{
		collectionLatencyHistogram: collectionLatencyHist,
		collectionsCounter:         collectionsCounter,
		collectionErrorsCounter:    collectionErrorsCounter,
	}
}

type GracefulCollectorWithMetrics struct {
	logger        *zap.Logger
	collectorName string
	collector     Interface

	collectionLatencyHistogram prometheus.Observer
	collectionsCounter         prometheus.Counter
	collectionErrorsCounter    prometheus.Counter
}

func NewGracefulCollectorWithMetrics(
	logger *zap.Logger,
	metrics *CollectProcessMetrics,
	collectorName string,
	collector Interface,
) *GracefulCollectorWithMetrics {
	metricLabels := prometheus.Labels{
		"collector_name": collectorName,
	}

	c := &GracefulCollectorWithMetrics{
		logger:        logger,
		collectorName: collectorName,
		collector:     collector,

		collectionLatencyHistogram: metrics.collectionLatencyHistogram.With(metricLabels),
		collectionsCounter:         metrics.collectionsCounter.With(metricLabels),
		collectionErrorsCounter:    metrics.collectionErrorsCounter.With(metricLabels),
	}

	return c
}

func (c *GracefulCollectorWithMetrics) Collect(ctx context.Context) error {
	t := time.Now()
	defer func() {
		c.collectionLatencyHistogram.Observe(time.Since(t).Seconds())
	}()

	c.collectionsCounter.Inc()

	c.logger.Debug("collection start", zap.String("collector", c.collectorName))

	err := c.collector.Collect(ctx)
	if err == nil {
		c.logger.Debug("collection finished", zap.String("collector", c.collectorName))
		return nil
	}

	c.collectionErrorsCounter.Inc()

	c.logger.Error("collection failed", zap.Error(err), zap.String("collector", c.collectorName))

	return nil
}
