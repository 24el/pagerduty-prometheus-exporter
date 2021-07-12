package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

func HTTPPrometheusMetrics(registerer prometheus.Registerer) mux.MiddlewareFunc {
	var (
		httpRequestsInflight = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: "http",
				Name:      "requests_inflight",
				Help:      "The number of inflight requests being handled at the same time.",
			},
			[]string{"handler"},
		)
		httpRequestDurHistogram = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: "http",
				Name:      "request_duration_seconds",
				Help:      "The latency of the HTTP requests.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"handler", "code", "method"},
		)
	)

	registerer.MustRegister(httpRequestsInflight)
	registerer.MustRegister(httpRequestDurHistogram)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			currentRoute := mux.CurrentRoute(r)
			if nil == currentRoute {
				next.ServeHTTP(w, r)

				return
			}

			routeName := currentRoute.GetName()
			if "" == routeName {
				next.ServeHTTP(w, r)

				return
			}

			var (
				code          = http.StatusOK
				headerWritten bool
				lock          sync.Mutex
				hooks         = httpsnoop.Hooks{
					WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
						return func(c int) {
							next(c)
							lock.Lock()
							defer lock.Unlock()
							if !headerWritten {
								code = c
								headerWritten = true
							}
						}
					},
				}
			)

			wi := httpsnoop.Wrap(w, hooks)

			// Measure inflights
			httpRequestsInflight.With(prometheus.Labels{"handler": routeName}).Inc()
			defer httpRequestsInflight.With(prometheus.Labels{"handler": routeName}).Dec()

			// Start the timer and when finishing measure the duration.
			start := time.Now()
			defer func() {
				duration := time.Since(start)

				httpRequestDurHistogram.With(prometheus.Labels{"handler": routeName, "code": strconv.Itoa(code), "method": r.Method}).Observe(duration.Seconds())
			}()

			next.ServeHTTP(wi, r)
		})
	}
}
