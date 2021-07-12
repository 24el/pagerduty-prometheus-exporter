package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	gopagerduty "github.com/PagerDuty/go-pagerduty"

	"github.com/24el/pagerduty-prometheus-exporter/internal/pagerduty"
)

const usersRequestLimit = 100

type UsersCollector struct {
	pdClient pagerduty.Client

	usersGauge *prometheus.GaugeVec
}

func NewUsersCollector(pdClient pagerduty.Client, registerer prometheus.Registerer) *UsersCollector {
	c := &UsersCollector{
		pdClient: pdClient,

		usersGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pagerduty_user",
			},
			[]string{
				"id",
				"name",
				"mail",
				"avatar",
				"color",
				"job_title",
				"role",
			},
		),
	}

	registerer.MustRegister(c.usersGauge)

	return c
}

func (c *UsersCollector) Collect(ctx context.Context) error {
	listOpts := gopagerduty.ListUsersOptions{}
	listOpts.Limit = usersRequestLimit

	for {
		list, err := c.pdClient.ListUsersWithContext(ctx, listOpts)
		if err != nil {
			return err
		}

		for _, user := range list.Users {
			c.usersGauge.With(prometheus.Labels{
				"id":        user.ID,
				"name":      user.Name,
				"mail":      user.Email,
				"avatar":    user.AvatarURL,
				"color":     user.Color,
				"job_title": user.JobTitle,
				"role":      user.Role,
			}).Set(1)
		}

		listOpts.Offset += list.Limit
		if !list.More {
			break
		}
	}

	return nil
}
