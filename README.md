# Pagerduty prometheus exporter

Pagerduty prometheus exporter for pagerduty analytics, users info, incident events(via pagerduty webhook v3 api)

## Configuration

```
Usage:
  pagerduty-prometheus-exporter [flags]

Flags:
      --analytics-report-periods durationSlice     scrape service analytic metric periods (default [2160h0m0s])
      --analytics-scrape-interval duration         scrape service analytic metric interval (default 1m0s)
      --analytics-service-metric-names strings     scrape service analytic metric names (default [total_escalation_count,total_incident_count,mean_seconds_to_resolve,mean_seconds_to_first_ack,up_time_pct])
      --debug                                      debug
      --dt-format string                           dt format (default "2006-01-02T15:04:05Z07:00")
  -h, --help                                       help for pagerduty-prometheus-exporter
      --incident-webhook-path string               incident webhook path (default "/v1/incidents")
      --incident-webhook-signature-secret string   incident webhook signature secret
      --metrics-prefix string                      metrics prefix
      --metrics-srv-port int                       metrics server port (default 9100)
      --pagerduty-auth-token string                pagerduty auth token
      --users-scrape-interval duration             scrape users interval (default 5m0s)
      --webhook-srv-port int                       webhook server port (default 8080)
```

## Metrics


| Metric                                         | Description                                                                                 |
|------------------------------------------------|---------------------------------------------------------------------------------------------|
| `pagerduty_service_{analytics_metric_name}`    | Collects service analytics from /analytics/metrics/incidents/services endpoint              |
| `pagerduty_incident_event`                     | Collects incident webhooks via v3 webhook pagerduty api                                     |
| `pagerduty_user`                               | Collects incident pagerduty users info from /users pagerduty endpoint                       |
| `pagerduty_metrics_collector_latency`          | Collection process latency                                                                  |
| `pagerduty_metrics_collector_collections_count`| Collection process count                                                                    |
| `pagerduty_metrics_collector_errors_count`     | Collection process errors count                                                             |

