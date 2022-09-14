package storage

import (
	"context"
	"sync"

	"github.com/ronny/slink/models"
)

type MemoryStorage struct {
	linkByID  *sync.Map
	linkByURL *sync.Map
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		linkByID:  &sync.Map{},
		linkByURL: &sync.Map{},
	}
}

func (s *MemoryStorage) Store(ctx context.Context, shortLink *models.ShortLink) error {
	// TODO: don't overwrite existing ShortLink with the same ID
	s.linkByID.Store(shortLink.ID, shortLink)
	s.linkByURL.Store(shortLink.LinkURL, shortLink)
	return nil
}

func (s *MemoryStorage) GetByID(ctx context.Context, shortLinkID string) (*models.ShortLink, error) {
	value, found := s.linkByID.Load(shortLinkID)
	if !found {
		return nil, nil
	}

	if shortLink, ok := value.(*models.ShortLink); ok {
		return shortLink, nil
	}

	return nil, nil
}

func (s *MemoryStorage) GetByURL(ctx context.Context, linkURL string) ([]*models.ShortLink, error) {
	value, found := s.linkByURL.Load(linkURL)
	if !found {
		return nil, nil
	}

	if shortLink, ok := value.(*models.ShortLink); ok {
		// can only store one link here for implementation simplicity
		return []*models.ShortLink{shortLink}, nil
	}

	return nil, nil
}