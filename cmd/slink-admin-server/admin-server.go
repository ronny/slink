package main

import (
	"context"
	"encoding/json"
	"errors"
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

type AdminServer struct {
	*http.Server
	router       *httprouter.Router
	svc          *slink.Slink
	slinkOptions []func(*slink.Slink)
	authKeys     []AuthKey
}

const (
	DefaultHandlerTimeoutDuration = 5 * time.Second
)

func NewAdminServer(ctx context.Context, options ...func(*AdminServer)) (*AdminServer, error) {
	s := &AdminServer{
		Server: &http.Server{
			WriteTimeout: 5 * time.Second,
			ReadTimeout:  5 * time.Second,
			IdleTimeout:  5 * time.Second,
		},
	}

	for _, option := range options {
		option(s)
	}

	if len(s.authKeys) == 0 {
		return nil, errors.New("missing authKeys, use WithAuthKeys to set at least one")
	}

	var err error
	s.svc, err = slink.NewSlink(ctx, s.slinkOptions...)
	if err != nil {
		return nil, fmt.Errorf("slink.NewSlink: %w", err)
	}

	s.router = httprouter.New()
	s.router.GET("/_live", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) { w.WriteHeader(http.StatusOK) })
	s.router.GET("/_ready", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) { w.WriteHeader(http.StatusOK) })
	s.apiRoute(http.MethodPost, "/get-or-create-short-link", s.handleGetOrCreateShortLink())
	s.apiRoute(http.MethodPost, "/create-short-link", s.handleCreateShortLink())
	s.apiRoute(http.MethodGet, "/short-link/:id", s.handleGetShortLink())
	s.Handler = s.router

	return s, nil
}

func (s *AdminServer) handleGetOrCreateShortLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var input slink.CreateInput
		err := decoder.Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		shortLink, err := s.svc.GetOrCreateShortLink(r.Context(), &input)
		if err != nil {
			var inverr *slink.ErrInvalidLinkURL
			if errors.As(err, &inverr) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(inverr.Error()))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		b, err := json.Marshal(shortLink)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func (s *AdminServer) handleCreateShortLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var input slink.CreateInput
		err := decoder.Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		shortLink, err := s.svc.CreateShortLink(r.Context(), &input)
		if err != nil {
			var inverr *slink.ErrInvalidLinkURL
			if errors.As(err, &inverr) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(inverr.Error()))
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		b, err := json.Marshal(shortLink)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func (s *AdminServer) handleGetShortLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		keyID := authKeyIDFromContext(ctx)
		log.Debug().Str("keyID", keyID).Msg("auth token matching")
		params := httprouter.ParamsFromContext(ctx)

		shortLinkID := params.ByName("id")

		shortLink, err := s.svc.GetShortLinkByID(r.Context(), shortLinkID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Error().Err(err).Msg("svc.GetShortLinkByID error, returning 500")

			return
		}

		if shortLink == nil {
			log.Debug().Str("shortLinkID", shortLinkID).Msg("handleGetShortLink: short link not found")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		b, err := json.Marshal(shortLink)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func (s *AdminServer) apiRoute(method, path string, h http.HandlerFunc) {
	labelsWithPath := prometheus.Labels{"path": path}

	s.router.Handler(
		method,
		path,
		s.requireAuthToken(
			http.TimeoutHandler(
				promhttp.InstrumentHandlerDuration(
					debug.IncomingRequestDurations().MustCurryWith(labelsWithPath),
					promhttp.InstrumentHandlerCounter(
						debug.IncomingRequests().MustCurryWith(labelsWithPath),
						h,
					),
				),
				DefaultHandlerTimeoutDuration,
				"timed out",
			).ServeHTTP,
		),
	)
}

func (s *AdminServer) Shutdown(ctx context.Context) error {
	s.SetKeepAlivesEnabled(false)
	return s.Server.Shutdown(ctx)
}

func WithListenAddr(addr string) func(*AdminServer) {
	return func(s *AdminServer) {
		s.Addr = addr
	}
}

func WithSlinkOptions(slinkOptions ...func(*slink.Slink)) func(*AdminServer) {
	return func(s *AdminServer) {
		s.slinkOptions = slinkOptions
	}
}

func WithAuthKeys(authKeys []AuthKey) func(*AdminServer) {
	return func(s *AdminServer) {
		s.authKeys = authKeys
	}
}
