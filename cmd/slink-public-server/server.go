package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/ronny/slink"
)

type PublicServer struct {
	*http.Server
	svc          *slink.Slink
	slinkOptions []func(*slink.Slink)
}

const DefaultTimeout = 5 * time.Second

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

	router := httprouter.New()
	router.GET("/:id", s.handleShortLinkLookup())
	s.Handler = router

	return s, nil
}

func (s *PublicServer) handleShortLinkLookup() httprouter.Handle {
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
