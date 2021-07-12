package webhook

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/24el/pagerduty-prometheus-exporter/internal/pagerduty"
)

type IncidentMetricsListener struct {
	dtFormat   string
	registerer prometheus.Registerer

	incidentEventGauge     *prometheus.GaugeVec
	incidentEventAssignees *prometheus.GaugeVec
	incidentEventTeams     *prometheus.GaugeVec
}

func NewIncidentMetricsListener(dtFormat string, registerer prometheus.Registerer) *IncidentMetricsListener {
	l := &IncidentMetricsListener{
		dtFormat:   dtFormat,
		registerer: registerer,
		incidentEventGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pagerduty_incident_event",
			},
			[]string{
				"incident_id",
				"type",
				"status",
				"event_type",
				"title",
				"service_id",
				"service_summary",
				"escalation_policy_id",
				"urgency",
				"priority_id",
				"dt",
			},
		),
		incidentEventAssignees: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pagerduty_incident_event_assignees",
			},
			[]string{"incident_id", "event_type", "assignee_id", "assignee_summary", "dt"},
		),
		incidentEventTeams: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "pagerduty_incident_event_teams",
			},
			[]string{"incident_id", "event_type", "team_id", "team_summary", "dt"},
		),
	}

	l.registerer.MustRegister(l.incidentEventGauge, l.incidentEventAssignees, l.incidentEventTeams)

	return l
}

func (l *IncidentMetricsListener) IncidentEventTriggered(event pagerduty.WebhookV3Event) error {
	switch event.EventType {
	case
		pagerduty.IncidentTriggeredEventType,
		pagerduty.IncidentAcknowledgedEventType,
		pagerduty.IncidentUnacknowledgedEventType,
		pagerduty.IncidentReassignedEventType,
		pagerduty.IncidentPriorityUpdatedEventType,
		pagerduty.IncidentDelegatedEventType,
		pagerduty.IncidentEscalatedEventType,
		pagerduty.IncidentReopenedEventType,
		pagerduty.IncidentResolvedEventType:
		return l.collectIncidentInfo(event)
	}

	return nil
}

func (l *IncidentMetricsListener) collectIncidentInfo(event pagerduty.WebhookV3Event) error {
	jsonData, err := json.Marshal(event.Data)
	if err != nil {
		return errors.Wrap(err, "marshal event data")
	}

	var incident pagerduty.WebhookV3Incident

	if err := json.Unmarshal(jsonData, &incident); err != nil {
		return errors.New("incident data must be with type popagerduty.Incident")
	}

	occurredAtFormatted := event.OccurredAt.Format(l.dtFormat)

	l.incidentEventGauge.With(prometheus.Labels{
		"incident_id":          incident.Id,
		"type":                 incident.Type,
		"status":               incident.Status,
		"event_type":           string(event.EventType),
		"title":                incident.Title,
		"service_id":           incident.Service.ID,
		"service_summary":      incident.Service.Summary,
		"escalation_policy_id": incident.EscalationPolicy.ID,
		"urgency":              incident.Urgency,
		"priority_id":          incident.Priority.ID,
		"dt":                   occurredAtFormatted,
	}).Set(1)

	for i := range incident.Assignees {
		l.incidentEventAssignees.With(prometheus.Labels{
			"incident_id":      incident.Id,
			"event_type":       string(event.EventType),
			"assignee_id":      incident.Assignees[i].ID,
			"assignee_summary": incident.Assignees[i].Summary,
			"dt":               occurredAtFormatted,
		}).Set(1)
	}

	for i := range incident.Teams {
		l.incidentEventTeams.With(prometheus.Labels{
			"incident_id":  incident.Id,
			"event_type":   string(event.EventType),
			"team_id":      incident.Teams[i].ID,
			"team_summary": incident.Teams[i].Summary,
			"dt":           occurredAtFormatted,
		}).Set(1)
	}

	return nil
}
