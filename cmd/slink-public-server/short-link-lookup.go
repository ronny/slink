package main

import (
	"context"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/ronny/slink/models"
	"github.com/rs/zerolog/log"
)

func (s *PublicServer) handleShortLinkLookup() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		params := httprouter.ParamsFromContext(ctx)

		shortLinkID := params.ByName("id")

		shortLink, err := s.svc.GetShortLinkByIDWithCache(r.Context(), shortLinkID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			log.Error().Err(err).Msg("svc.GetShortLinkByID error, returning 500")

			return
		}

		if shortLink == nil {
			if s.fallbackRedirectURL != "" {
				w.Header().Add("Location", s.fallbackRedirectURL)
				w.WriteHeader(http.StatusTemporaryRedirect)
				go s.trackShortLinkLookup(shortLinkID, shortLink, r, http.StatusTemporaryRedirect, s.fallbackRedirectURL)
				return
			}

			w.WriteHeader(http.StatusNotFound)
			go s.trackShortLinkLookup(shortLinkID, shortLink, r, http.StatusNotFound, "")
			return
		}

		if shortLink.Expired() {
			if s.fallbackRedirectURL != "" {
				w.Header().Add("Location", s.fallbackRedirectURL)
				w.WriteHeader(http.StatusTemporaryRedirect)
				go s.trackShortLinkLookup(shortLinkID, shortLink, r, http.StatusTemporaryRedirect, "")
				return
			}

			w.WriteHeader(http.StatusGone)
			go s.trackShortLinkLookup(shortLinkID, shortLink, r, http.StatusGone, "")
			return
		}

		w.Header().Add("Location", shortLink.LinkURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
		go s.trackShortLinkLookup(shortLinkID, shortLink, r, http.StatusTemporaryRedirect, shortLink.LinkURL)
	}
}

func (s *PublicServer) trackShortLinkLookup(
	shortLinkID string,
	shortLink *models.ShortLink,
	r *http.Request,
	responseStatusCode int,
	responseLocation string,
) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCtx()

	if s.tracker == nil {
		return
	}

	payload, err := s.payloadBuilder.BuildShortLinkLookupPayload(ctx, shortLinkID, shortLink, r, responseStatusCode, responseLocation)
	if err != nil {
		log.Error().
			Err(err).
			Str("shortLinkID", shortLinkID).
			Msg("failed to build payload to track lookup")
	}

	err = s.tracker.TrackShortLinkLookupRequest(ctx, payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("shortLinkID", shortLinkID).
			Msg("failed to track lookup")
	}
}
