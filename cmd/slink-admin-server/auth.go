package main

import (
	"context"
	"net/http"
	"regexp"
)

type AuthKey struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

func (s *AdminServer) requireAuthToken(h http.HandlerFunc) http.HandlerFunc {
	idsByToken := make(map[string]string)
	for _, authKey := range s.authKeys {
		idsByToken[authKey.Token] = authKey.ID
	}

	return func(w http.ResponseWriter, r *http.Request) {
		authHeaderVal := r.Header.Get("Authorization")
		token := bearerPrefixRe.ReplaceAllString(authHeaderVal, "")
		id := idsByToken[token]

		if id == "" {
			http.Error(w, "insert coin", http.StatusUnauthorized)
			return
		}

		h(w, r.WithContext(context.WithValue(r.Context(), ctxKey, id)))
	}
}

var bearerPrefixRe = regexp.MustCompile(`(?i)^Bearer\s+`)

type keyIDCtxKey struct{}

var ctxKey = keyIDCtxKey{}

func authKeyIDFromContext(reqCtx context.Context) string {
	val := reqCtx.Value(ctxKey)
	if id, ok := val.(string); ok {
		return id
	}
	return ""
}
