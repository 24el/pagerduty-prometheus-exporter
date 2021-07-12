package collector

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type PeriodicCollector struct {
	collectInterval time.Duration
	collectors      []Interface
}

func NewPeriodicCollector(interval time.Duration, collectors ...Interface) *PeriodicCollector {
	return &PeriodicCollector{
		collectInterval: interval,
		collectors:      collectors,
	}
}

func (c *PeriodicCollector) Collect(ctx context.Context) error {
	eg, gCtx := errgroup.WithContext(ctx)

	for i := range c.collectors {
		collector := c.collectors[i]

		eg.Go(func() error {
			if err := collector.Collect(ctx); err != nil {
				return err
			}

			ticker := time.NewTicker(c.collectInterval)

			for {
				select {
				case <-gCtx.Done():
					return nil
				case <-ticker.C:
					if err := collector.Collect(ctx); err != nil {
						return err
					}
				}
			}
		})
	}

	return eg.Wait()
}
