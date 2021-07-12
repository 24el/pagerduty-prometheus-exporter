package pagerduty

import (
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

type WebhookEventType string

const (
	IncidentAcknowledgedEventType     = "incident.acknowledged"
	IncidentAnnotatedEventType        = "incident.annotated"
	IncidentDelegatedEventType        = "incident.delegated"
	IncidentEscalatedEventType        = "incident.escalated"
	IncidentPriorityUpdatedEventType  = "incident.priority_updated"
	IncidentReassignedEventType       = "incident.reassigned"
	IncidentReopenedEventType         = "incident.reopened"
	IncidentResolvedEventType         = "incident.resolved"
	IncidentResponderAddedEventType   = "incident.responder.added"
	IncidentResponderRepliedEventType = "incident.responder.replied"
	IncidentTriggeredEventType        = "incident.triggered"
	IncidentUnacknowledgedEventType   = "incident.unacknowledged"
)

type WebhookV3 struct {
	Event WebhookV3Event `json:"event"`
}

type WebhookV3Client struct {
	Name string `json:"name"`
}

type WebhookV3Event struct {
	ID           string           `json:"id"`
	EventType    WebhookEventType `json:"event_type"`
	ResourceType string           `json:"resource_type"`
	OccurredAt   time.Time        `json:"occurred_at"`
	Agent        pagerduty.Agent  `json:"agent"`
	Client       WebhookV3Client  `json:"client"`
	Data         interface{}      `json:"data"`
}

type WebhookV3Incident struct {
	Number           uint                       `json:"number,omitempty"`
	Title            string                     `json:"title,omitempty"`
	Type             string                     `json:"type,omitempty"`
	Self             string                     `json:"self,omitempty"`
	HTMLURL          string                     `json:"html_url,omitempty"`
	Service          pagerduty.APIObject        `json:"service,omitempty"`
	Assignees        []pagerduty.APIObject      `json:"assignees,omitempty"`
	EscalationPolicy pagerduty.APIObject        `json:"escalation_policy,omitempty"`
	Teams            []pagerduty.APIObject      `json:"teams,omitempty"`
	Priority         pagerduty.APIObject        `json:"priority,omitempty"`
	Urgency          string                     `json:"urgency,omitempty"`
	Status           string                     `json:"status,omitempty"`
	Id               string                     `json:"id,omitempty"`
	ConferenceBridge pagerduty.ConferenceBridge `json:"conference_bridge,omitempty"`
}
