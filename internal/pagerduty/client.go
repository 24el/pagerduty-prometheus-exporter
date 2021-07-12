package pagerduty

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
)

type Client interface {
	QueryMetricReport(ctx context.Context, params ServiceMetricReportParams) (*Report, error)
	ListUsersWithContext(ctx context.Context, o pagerduty.ListUsersOptions) (*pagerduty.ListUsersResponse, error)
}
