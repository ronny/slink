package debug

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RoundTripperPathFunc func(r *http.Request) string

func DefaultRoundTripperPathFunc(r *http.Request) string {
	return r.URL.Path
}

// NewRoundTripper creates an http.RoundTripper to instrument outgoing request
// counts and durations. It's basically a combo of NewCounterRoundTripper and
// NewDurationRoundTripper.
func NewRoundTripper(original http.RoundTripper, pathFunc RoundTripperPathFunc) http.RoundTripper {
	if original == nil {
		original = http.DefaultTransport
	}
	return NewCounterRoundTripper(NewDurationRoundTripper(original, pathFunc), pathFunc)
}

// NewCounterRoundTripper creates an http.RoundTripper to instrument outgoing
// request counts.
func NewCounterRoundTripper(original http.RoundTripper, pathFunc RoundTripperPathFunc) http.RoundTripper {
	if original == nil {
		original = http.DefaultTransport
	}
	if pathFunc == nil {
		pathFunc = DefaultRoundTripperPathFunc
	}
	return promhttp.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := original.RoundTrip(r)
		if resp != nil {
			globalMetrics.outgoingRequests.WithLabelValues(
				r.URL.Host,
				pathFunc(r),
				strconv.Itoa(resp.StatusCode),
			).Inc()
		}
		return resp, err
	})
}

// NewDurationRoundTripper creates an http.RoundTripper to instrument outgoing
// request durations.
func NewDurationRoundTripper(original http.RoundTripper, pathFunc RoundTripperPathFunc) http.RoundTripper {
	if original == nil {
		original = http.DefaultTransport
	}
	if pathFunc == nil {
		pathFunc = DefaultRoundTripperPathFunc
	}
	return promhttp.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := original.RoundTrip(r)
		if resp != nil {
			globalMetrics.outgoingRequestDurations.WithLabelValues(
				r.URL.Host,
				pathFunc(r),
				strconv.Itoa(resp.StatusCode),
			).Observe(time.Since(start).Seconds())
		}
		return resp, err
	})
}
