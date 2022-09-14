package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/ronny/slink"
)

type AdminServer struct {
	*http.Server
	svc          *slink.Slink
	slinkOptions []func(*slink.Slink)
}

const DefaultTimeout = 5 * time.Second

func NewAdminServer(ctx context.Context, options ...func(*AdminServer)) (*AdminServer, error) {
	s := &AdminServer{
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

	router := httprouter.New()
	router.POST("/short-links", s.handleCreateShortLink())
	router.GET("/short-links/:id", s.handleGetShortLink())
	s.Handler = router

	return s, nil
}

func (s *AdminServer) handleCreateShortLink() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		decoder := json.NewDecoder(r.Body)
		var input slink.ShortenInput
		err := decoder.Decode(&input)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		shortLink, err := s.svc.Shorten(r.Context(), &input)
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

func (s *AdminServer) handleGetShortLink() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		shortLinkID := params.ByName("id")

		shortLink, err := s.svc.GetShortLinkByID(r.Context(), shortLinkID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			// TODO: optional logger/error reporter accepted in NewPublicServer?
			log.Printf("svc.GetShortLinkByID: %v — returning 500", err)

			return
		}

		if shortLink == nil {
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

func WithListenAddr(addr string) func(*AdminServer) {
	return func(ps *AdminServer) {
		ps.Addr = addr
	}
}

func WithSlinkOptions(slinkOptions ...func(*slink.Slink)) func(*AdminServer) {
	return func(ps *AdminServer) {
		ps.slinkOptions = slinkOptions
	}
}
