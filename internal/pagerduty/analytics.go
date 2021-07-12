package pagerduty

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type ReportMetricName string

const (
	ReportMetricMeanAssignmentCount            ReportMetricName = "mean_assignment_count"
	ReportMetricMeanEngagedSeconds             ReportMetricName = "mean_engaged_seconds"
	ReportMetricMeanEngagedUserCount           ReportMetricName = "mean_engaged_user_count"
	ReportMetricMeanSecondsToEngage            ReportMetricName = "mean_seconds_to_engage"
	ReportMetricMeanSecondsToFirstAck          ReportMetricName = "mean_seconds_to_first_ack"
	ReportMetricMeanSecondsToResolve           ReportMetricName = "mean_seconds_to_resolve"
	ReportMetricMeanSecondsToMobilize          ReportMetricName = "mean_seconds_to_mobilize"
	ReportMetricTotalBusinessHourInterruptions ReportMetricName = "total_business_hour_interruptions"
	ReportMetricTotalEngagedSeconds            ReportMetricName = "total_engaged_seconds"
	ReportMetricTotalEscalationsCount          ReportMetricName = "total_escalation_count"
	ReportMetricIncidentCount                  ReportMetricName = "total_incident_count"
	ReportMetricTotalOffHourInterruptions      ReportMetricName = "total_off_hour_interruptions"
	ReportMetricTotalSleepHourInterruptions    ReportMetricName = "total_sleep_hour_interruptions"
	ReportMetricTotalSnoozedSeconds            ReportMetricName = "total_snoozed_seconds"
	ReportMetricIncidentUpTimePct              ReportMetricName = "up_time_pct"
)

func GetReportMetricName(mn string) (ReportMetricName, error) {
	switch ReportMetricName(mn) {
	case ReportMetricMeanAssignmentCount,
		ReportMetricMeanEngagedSeconds,
		ReportMetricMeanEngagedUserCount,
		ReportMetricMeanSecondsToEngage,
		ReportMetricMeanSecondsToFirstAck,
		ReportMetricMeanSecondsToResolve,
		ReportMetricMeanSecondsToMobilize,
		ReportMetricTotalBusinessHourInterruptions,
		ReportMetricTotalEngagedSeconds,
		ReportMetricTotalEscalationsCount,
		ReportMetricIncidentCount,
		ReportMetricTotalOffHourInterruptions,
		ReportMetricTotalSleepHourInterruptions,
		ReportMetricTotalSnoozedSeconds,
		ReportMetricIncidentUpTimePct:
		return ReportMetricName(mn), nil
	}

	return "", fmt.Errorf("metric %s not found", mn)
}

type ReportTime time.Time

func (t ReportTime) MarshalJSON() ([]byte, error) {
	ft := time.Time(t).Format("2006-01-02T15:04:05.999")

	return []byte(fmt.Sprintf("\"%s\"", ft)), nil
}

type ServiceMetricReportFilters struct {
	CreatedAtStart ReportTime `json:"created_at_start"`
	CreatedAtEnd   ReportTime `json:"created_at_end"`
}

type ServiceMetricReportParams struct {
	TimeZone string                     `json:"time_zone"`
	Filters  ServiceMetricReportFilters `json:"filters"`
}

type ReportItem struct {
	MeanAssignmentCount            float64 `json:"mean_assignment_count"`
	MeanEngagedSeconds             float64 `json:"mean_engaged_seconds"`
	MeanEngagedUserCount           float64 `json:"mean_engaged_user_count"`
	MeanSecondsToEngage            float64 `json:"mean_seconds_to_engage"`
	MeanSecondsToFirstAck          float64 `json:"mean_seconds_to_first_ack"`
	MeanSecondsToMobilize          float64 `json:"mean_seconds_to_mobilize"`
	MeanSecondsToResolve           float64 `json:"mean_seconds_to_resolve"`
	RangeStart                     string  `json:"range_start"`
	ServiceID                      string  `json:"service_id"`
	ServiceName                    string  `json:"service_name"`
	TotalBusinessHourInterruptions float64 `json:"total_business_hour_interruptions"`
	TotalEngagedSeconds            float64 `json:"total_engaged_seconds"`
	TotalEscalationCount           float64 `json:"total_escalation_count"`
	TotalIncidentCount             float64 `json:"total_incident_count"`
	TotalOffHourInterruptions      float64 `json:"total_off_hour_interruptions"`
	TotalSleepHourInterruptions    float64 `json:"total_sleep_hour_interruptions"`
	TotalSnoozedSeconds            float64 `json:"total_snoozed_seconds"`
	UpTimePct                      float64 `json:"up_time_pct"`
}

func (i *ReportItem) GetMetricByName(mn ReportMetricName) (float64, error) {
	switch mn {
	case ReportMetricIncidentCount:
		return i.TotalIncidentCount, nil
	case ReportMetricMeanSecondsToFirstAck:
		return i.MeanSecondsToFirstAck, nil
	case ReportMetricMeanSecondsToResolve:
		return i.MeanSecondsToResolve, nil
	case ReportMetricTotalEscalationsCount:
		return i.TotalEscalationCount, nil
	case ReportMetricIncidentUpTimePct:
		return i.UpTimePct, nil
	case ReportMetricMeanAssignmentCount:
		return i.MeanAssignmentCount, nil
	case ReportMetricMeanEngagedSeconds:
		return i.MeanEngagedSeconds, nil
	case ReportMetricMeanEngagedUserCount:
		return i.MeanEngagedUserCount, nil
	case ReportMetricMeanSecondsToEngage:
		return i.MeanSecondsToEngage, nil
	case ReportMetricMeanSecondsToMobilize:
		return i.MeanSecondsToMobilize, nil
	case ReportMetricTotalBusinessHourInterruptions:
		return i.TotalBusinessHourInterruptions, nil
	case ReportMetricTotalEngagedSeconds:
		return i.TotalEngagedSeconds, nil
	case ReportMetricTotalOffHourInterruptions:
		return i.TotalOffHourInterruptions, nil
	case ReportMetricTotalSleepHourInterruptions:
		return i.TotalSleepHourInterruptions, nil
	case ReportMetricTotalSnoozedSeconds:
		return i.TotalSnoozedSeconds, nil
	}

	return 0, fmt.Errorf("metric %s not found", mn)
}

type Report struct {
	Data []ReportItem `json:"data"`
}

func (c *ExtendedClient) QueryMetricReport(
	ctx context.Context,
	params ServiceMetricReportParams,
) (*Report, error) {
	resp, err := c.post(
		ctx,
		"/analytics/metrics/incidents/services",
		params,
		map[string]string{"X-EARLY-ACCESS": "analytics-v2"},
	)
	return c.getMetricReportFromResponse(resp, err)
}

func (c *ExtendedClient) getMetricReportFromResponse(resp *http.Response, err error) (*Report, error) {
	if err != nil {
		return nil, err
	}

	var target Report
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("could not decode JSON response: %v", dErr)
	}

	return &target, nil
}
