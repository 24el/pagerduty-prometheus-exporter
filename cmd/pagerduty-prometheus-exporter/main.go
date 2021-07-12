package main

import (
	"log"

	"github.com/24el/pagerduty-prometheus-exporter/cmd/pagerduty-prometheus-exporter/cmd"
)

func main() {
	if err := cmd.NewPagerdutyPrometheusExporterCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
