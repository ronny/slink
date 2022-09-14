package storage

import (
	"context"

	"github.com/ronny/slink/models"
)

type Storage interface {
	Store(ctx context.Context, shortLink *models.ShortLink) error
	GetByID(ctx context.Context, shortLinkID string) (*models.ShortLink, error)
	GetByURL(ctx context.Context, linkURL string) ([]*models.ShortLink, error) // TODO: pagination of output
}
