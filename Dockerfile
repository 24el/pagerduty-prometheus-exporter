ARG GOVERSION=1.16
FROM golang:${GOVERSION}-alpine as builder

ARG GOARCH
ENV GOARCH=${GOARCH}

WORKDIR /go/src/github.com/24el/pagerduty-prometheus-exporter/

COPY . /go/src/github.com/24el/pagerduty-prometheus-exporter/

RUN go build -o /pagerduty-prometheus-exporter ./cmd/pagerduty-prometheus-exporter

FROM alpine:3.12

WORKDIR /var/lib/pagerduty-prometheus-exporter

COPY --from=builder /pagerduty-prometheus-exporter /usr/bin/

ENTRYPOINT ["/usr/bin/pagerduty-prometheus-exporter"]