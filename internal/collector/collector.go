package collector

import "context"

type Interface interface {
	Collect(ctx context.Context) error
}
