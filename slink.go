package slink

import (
	"context"
	"errors"
	"fmt"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ronny/slink/ids"
	"github.com/ronny/slink/models"
	"github.com/ronny/slink/storage"
	"github.com/rs/zerolog/log"
)

type Slink struct {
	idgen             ids.Generator
	storage           storage.Storage
	lruCache          *lru.Cache
	maxCreateAttempts int
}

type CreateInput struct {
	LinkURL   string `json:"linkUrl"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// GetOrCreateShortLink returns an existing ShortLink if both LinkURL and ExpiresAt match the input,
// or creates a new ShortLink if no matching ShortLink can be found.
//
// No normalisation is done on LinkURL. `https://example.com?a=1&b=2` is considered different to `https://example.com?b=2&a1`.
// Only strict exact string match is used for lookups (whatever's supported by the storage backend).
func (s *Slink) GetOrCreateShortLink(ctx context.Context, input *CreateInput) (*models.ShortLink, error) {
	// TODO: pagination
	shortLinks, err := s.GetShortLinksByURL(ctx, input.LinkURL)
	if err != nil {
		return nil, err
	}

	var matchingShortLink *models.ShortLink
	for _, shortLink := range shortLinks {
		if shortLink.LinkURL == input.LinkURL && shortLink.ExpiresAt == input.ExpiresAt {
			matchingShortLink = shortLink
			break
		}
	}

	if matchingShortLink != nil {
		return matchingShortLink, nil
	}

	return s.CreateShortLink(ctx, input)
}

// CreateShortLink unconditionally creates a new ShortLink, even when one with the exact same LinkURL already exists
func (s *Slink) CreateShortLink(ctx context.Context, input *CreateInput) (*models.ShortLink, error) {
	if input == nil {
		return nil, errors.New("input is nil (BUG?)")
	}

	if input.LinkURL == "" {
		return nil, &ErrInvalidLinkURL{msg: "link URL must not be empty"}
	}

	for attempt := 1; attempt <= s.maxCreateAttempts; attempt++ {
		id, err := s.idgen.GenerateID()
		if err != nil {
			return nil, err
		}

		shortLink := &models.ShortLink{
			ID:        id,
			LinkURL:   input.LinkURL,
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
			ExpiresAt: input.ExpiresAt,
		}
		err = s.storage.Create(ctx, shortLink)
		if err != nil {
			var ex *storage.ErrShortLinkAlreadyExists
			if errors.As(err, &ex) {
				log.Info().Err(ex).Int("attempt", attempt).Str("id", id).Msg("short link ID collision, retrying...")
				continue
			}
			return nil, fmt.Errorf("storage.Create: %w", err)
		} else {
			return shortLink, nil
		}
	}

	return nil, &ErrCreateAttemptsExhausted{attempts: s.maxCreateAttempts}
}

// GetShortLinkByIDWithCache looks up ShortLink by the given ID from the LRU
// cache first, if found it returns it, otherwise it looks the ShortLink up in
// the storage, and returns it if it's found in the storage, adding it to the
// LRU cache if found, or it returns nil otherwise.
func (s *Slink) GetShortLinkByIDWithCache(ctx context.Context, shortLinkID string) (*models.ShortLink, error) {
	if shortLinkID == "" {
		return nil, &ErrInvalidShortLinkID{msg: "short link ID must not be empty"}
	}

	if s.lruCache != nil {
		if item, found := s.lruCache.Get(shortLinkID); found {
			if shortLink, ok := item.(*models.ShortLink); ok {
				return shortLink, nil
			}
			log.Warn().Msgf("found item in LRU cache but failed to assert as *models.ShortLink, ignoring: %+v", item)
		}
	}

	shortLink, err := s.GetShortLinkByID(ctx, shortLinkID)
	if err != nil {
		return nil, fmt.Errorf("storage.Get: %w", err)
	}

	if s.lruCache != nil {
		evicted := s.lruCache.Add(shortLinkID, shortLink)
		log.Debug().Bool("evicted", evicted).Msg("added shortLink to LRU cache")
	}

	return shortLink, nil
}

// GetShortLinkByID looks up a ShortLink by its ID, returing it if found, or nil otherwise.
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

const DefaultMaxCreateAttempts = 3

func NewSlink(ctx context.Context, options ...func(*Slink)) (*Slink, error) {
	lruCache, err := lru.New(1000)
	if err != nil {
		return nil, fmt.Errorf("lru.New: %w", err)
	}

	s := &Slink{
		lruCache:          lruCache,
		maxCreateAttempts: DefaultMaxCreateAttempts,
	}

	for _, option := range options {
		option(s)
	}

	if s.maxCreateAttempts < 1 {
		return nil, errors.New("maxCreateAttempts must be at least 1")
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

func WithMaxCreateAttempts(maxCreateAttempts int) func(*Slink) {
	return func(s *Slink) {
		s.maxCreateAttempts = maxCreateAttempts
	}
}
