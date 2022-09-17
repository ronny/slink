package storage

import (
	"context"
	"fmt"

	"github.com/ronny/slink/models"
)

type Storage interface {
	Create(ctx context.Context, shortLink *models.ShortLink) error
	GetByID(ctx context.Context, shortLinkID string) (*models.ShortLink, error)
	GetByURL(ctx context.Context, linkURL string) ([]*models.ShortLink, error) // TODO: pagination of output
}

type ErrShortLinkAlreadyExists struct {
	ShortLinkID string
}

func (e *ErrShortLinkAlreadyExists) Error() string {
	return fmt.Sprintf("ErrShortLinkAlreadyExists: ShortLink with ID %s already exists", e.ShortLinkID)
}
