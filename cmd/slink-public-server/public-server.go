package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ronny/slink"
	"github.com/ronny/slink/debug"
	"github.com/rs/zerolog/log"
)

type PublicServer struct {
	*http.Server
	router       *httprouter.Router
	svc          *slink.Slink
	slinkOptions []func(*slink.Slink)
}

const (
	DefaultHandlerTimeoutDuration = 5 * time.Second
)

func NewPublicServer(ctx context.Context, options ...func(*PublicServer)) (*PublicServer, error) {
	s := &PublicServer{
		Server: &http.Server{
			WriteTimeout: 1 * time.Second,
			ReadTimeout:  1 * time.Second,
			IdleTimeout:  1 * time.Second,
		},
	}

	for _, option := range options {
		option(s)
	}

	var err error
	s.svc, err = slink.NewSlink(ctx, s.slinkOptions...)
	if err != nil {
		return nil, fmt.Errorf("slink.NewSlink: %w", err)
	}

	s.router = httprouter.New()
	s.apiRoute(http.MethodGet, "/:id", s.handleShortLinkLookup())
	s.Handler = s.router

	return s, nil
}

func (s *PublicServer) apiRoute(method, path string, handler http.Handler) {
	labelsWithPath := prometheus.Labels{"path": path}

	s.router.Handler(
		method,
		path,
		http.TimeoutHandler(
			promhttp.InstrumentHandlerDuration(
				debug.IncomingRequestDurations().MustCurryWith(labelsWithPath),
				promhttp.InstrumentHandlerCounter(
					debug.IncomingRequests().MustCurryWith(labelsWithPath),
					handler,
				),
			),
			DefaultHandlerTimeoutDuration,
			"timed out",
		),
	)
}

func (s *PublicServer) handleShortLinkLookup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		params := httprouter.ParamsFromContext(ctx)

		shortLinkID := params.ByName("id")

		shortLink, err := s.svc.GetShortLinkByID(r.Context(), shortLinkID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Error().Err(err).Msg("svc.GetShortLinkByID error, returning 500")

			return
		}

		if shortLink == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if shortLink.Expired() {
			w.WriteHeader(http.StatusGone)
			return
		}

		w.Header().Add("Location", shortLink.LinkURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func WithListenAddr(addr string) func(*PublicServer) {
	return func(ps *PublicServer) {
		ps.Addr = addr
	}
}

func WithSlinkOptions(slinkOptions ...func(*slink.Slink)) func(*PublicServer) {
	return func(ps *PublicServer) {
		ps.slinkOptions = slinkOptions
	}
}
