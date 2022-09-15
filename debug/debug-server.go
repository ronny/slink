package debug

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// DebugServer serves the prometheus metrics over http so that prometheus can scrape the metrics.
type DebugServer struct {
	*http.Server
}

func NewDebugServer(listenAddr string) (*DebugServer, error) {
	if listenAddr == "" {
		return nil, errors.New("listenAddr is missing")
	}
	return &DebugServer{
		Server: &http.Server{
			Addr:         listenAddr,
			WriteTimeout: 60 * time.Second,
			ReadTimeout:  60 * time.Second,
			IdleTimeout:  60 * time.Second,
			Handler:      newRouter(),
		},
	}, nil
}

func (s *DebugServer) Shutdown(ctx context.Context) error {
	s.SetKeepAlivesEnabled(false)
	return s.Server.Shutdown(ctx)
}

func newRouter() *httprouter.Router {
	router := httprouter.New()
	router.HandleMethodNotAllowed = false
	router.PanicHandler = panicHandler()
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())
	router.HandlerFunc(http.MethodGet, "/debug/pprof/", pprof.Index)
	router.Handler(http.MethodGet, "/debug/pprof/allocs", pprof.Handler("allocs"))
	router.Handler(http.MethodGet, "/debug/pprof/block", pprof.Handler("block"))
	router.HandlerFunc(http.MethodGet, "/debug/pprof/cmdline", pprof.Cmdline)
	router.Handler(http.MethodGet, "/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handler(http.MethodGet, "/debug/pprof/heap", pprof.Handler("heap"))
	router.Handler(http.MethodGet, "/debug/pprof/mutex", pprof.Handler("mutex"))
	router.HandlerFunc(http.MethodGet, "/debug/pprof/profile", pprof.Profile)
	router.Handler(http.MethodGet, "/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	router.HandlerFunc(http.MethodGet, "/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc(http.MethodPost, "/debug/pprof/symbol", pprof.Symbol)
	router.HandlerFunc(http.MethodGet, "/debug/pprof/trace", pprof.Trace)
	return router
}

func panicHandler() func(http.ResponseWriter, *http.Request, interface{}) {
	return func(w http.ResponseWriter, r *http.Request, panicPayload interface{}) {
		switch typedPayload := panicPayload.(type) {
		case error:
			log.Error().Err(typedPayload).Msg("handled panic")
		default:
			log.Error().Str("panicPayload", fmt.Sprintf("%+v", typedPayload)).Msg("handled panic")
		}
		w.WriteHeader(http.StatusInternalServerError)
	}
}
