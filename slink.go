package slink

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ronny/slink/ids"
	"github.com/ronny/slink/models"
	"github.com/ronny/slink/storage"
)

type Slink struct {
	idgen   ids.Generator
	storage storage.Storage
}

type ShortenInput struct {
	LinkURL   string `json:"linkUrl"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

func (s *Slink) Shorten(ctx context.Context, input *ShortenInput) (*models.ShortLink, error) {
	if input == nil {
		return nil, errors.New("input is nil (BUG?)")
	}

	if input.LinkURL == "" {
		return nil, &ErrInvalidLinkURL{msg: "link URL must not be empty"}
	}

	shortLink := &models.ShortLink{
		ID:        s.idgen.GenerateID(),
		LinkURL:   input.LinkURL,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: input.ExpiresAt,
	}

	err := s.storage.Store(ctx, shortLink)
	if err != nil {
		return nil, fmt.Errorf("storage.Store: %w", err)
	}

	return shortLink, nil
}

func (s *Slink) GetShortLinkByID(ctx context.Context, shortLinkID string) (*models.ShortLink, error) {
	if shortLinkID == "" {
		return nil, &ErrInvalidShortLinkID{msg: "short link ID must not be empty"}
	}

	shortLink, err := s.storage.GetByID(ctx, shortLinkID)
	if err != nil {
		return nil, fmt.Errorf("storage.Get: %w", err)
	}

	return shortLink, nil
}

func (s *Slink) GetShortLinksByURL(ctx context.Context, linkURL string) ([]*models.ShortLink, error) {
	if linkURL == "" {
		return nil, &ErrInvalidLinkURL{msg: "link URL must not be empty"}
	}

	shortLinks, err := s.storage.GetByURL(ctx, linkURL)
	if err != nil {
		return nil, fmt.Errorf("storage.Get: %w", err)
	}

	return shortLinks, nil
}

func NewSlink(ctx context.Context, options ...func(*Slink)) (*Slink, error) {
	s := &Slink{}

	for _, option := range options {
		option(s)
	}

	if s.idgen == nil {
		var err error
		s.idgen, err = ids.NewNanoIDGenerator()
		if err != nil {
			return nil, fmt.Errorf("ids.NewNanoIDGenerator: %w", err)
		}
	}

	if s.storage == nil {
		var err error
		s.storage, err = storage.NewDynamoDBStorage(ctx)
		if err != nil {
			return nil, fmt.Errorf("storage.NewDynamoDBStorage: %w", err)
		}
	}

	return s, nil
}

func WithIDGenerator(idgen ids.Generator) func(*Slink) {
	return func(s *Slink) {
		s.idgen = idgen
	}
}

func WithStorage(storage storage.Storage) func(*Slink) {
	return func(s *Slink) {
		s.storage = storage
	}
}
