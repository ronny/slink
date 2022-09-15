package debug

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const Namespace = "slink"

type Metrics struct {
	incomingRequests         *prometheus.CounterVec
	incomingRequestDurations *prometheus.HistogramVec
	shortLinkCreations       *prometheus.CounterVec
	deniedShortLinkIDs       *prometheus.CounterVec
	redirects                *prometheus.CounterVec
}

var globalMetrics *Metrics

func init() {
	// Disable the default built-in Go metrics. They're useful but expensive to store in Prometheus.
	prometheus.Unregister(collectors.NewGoCollector())

	globalMetrics = &Metrics{
		incomingRequests: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "incoming_requests_total",
		}, []string{"code", "path"}),
		incomingRequestDurations: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: Namespace,
			Name:      "incoming_request_duration_seconds",
		}, []string{"path"}),
		shortLinkCreations: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "short_link_creations_total",
		}, []string{}),
		deniedShortLinkIDs: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "denied_short_link_ids_total",
			Help:      "total number of generated short link ids that were denied because they matched denylist",
		}, []string{}),
		redirects: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: Namespace,
			Name:      "redirects_total",
		}, []string{}),
	}
}

func IncomingRequests() *prometheus.CounterVec {
	return globalMetrics.incomingRequests
}

func IncomingRequestDurations() *prometheus.HistogramVec {
	return globalMetrics.incomingRequestDurations
}
