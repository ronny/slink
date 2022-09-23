package tracking

import (
	"context"
)

type Tracker interface {
	TrackShortLinkLookupRequest(ctx context.Context, payload *ShortLinkLookupPayload) error
}
