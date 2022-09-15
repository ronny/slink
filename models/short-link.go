package models

import (
	"time"

	"github.com/rs/zerolog/log"
)

type ShortLink struct {
	ID        string `json:"id" dynamodbav:"id"`
	LinkURL   string `json:"linkUrl" dynamodbav:"linkUrl"`
	CreatedAt string `json:"createdAt" dynamodbav:"createdAt"`
	ExpiresAt string `json:"expiresAt,omitempty" dynamodbav:"expiresAt,omitempty"`
}

func (sl *ShortLink) Expired() bool {
	if sl.ExpiresAt == "" {
		return false
	}

	expiry, err := time.Parse(time.RFC3339, sl.ExpiresAt)
	if err != nil {
		log.Warn().Err(err).Str("ExpiresAt", sl.ExpiresAt).Msg("time.Parse ExpiresAt failed, assuming there's no expiry")
		return false
	}

	return expiry.After(time.Now().UTC())
}
