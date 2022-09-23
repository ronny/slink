package tracking

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ronny/slink/models"
)

type ShortLinkLookupPayload struct {
	ShortLinkID        string            `json:"shortLinkId"`
	ShortLinkFound     bool              `json:"shortLinkFound"`
	ShortLinkExpired   bool              `json:"shortLinkExpired"`
	TargetURL          string            `json:"targetUrl"` // not necessarily the same as the actual redirect URL (`ResponseLocation`)
	RequestHost        string            `json:"requestHost"`
	RequestHeaders     map[string]string `json:"requestHeaders"`
	RequestedAt        string            `json:"requestedAt"`
	ResponseStatusCode int               `json:"responseStatusCode"`
	ResponseLocation   string            `json:"responseLocation"`
}

type PayloadBuilder struct {
	trustedHeaders []string
}

func NewPayloadBuilder(trustedHeaders []string) *PayloadBuilder {
	return &PayloadBuilder{
		trustedHeaders: trustedHeaders,
	}
}

func (pb *PayloadBuilder) BuildShortLinkLookupPayload(
	ctx context.Context,
	shortLinkID string,
	shortLink *models.ShortLink,
	r *http.Request,
	responseStatusCode int,
	responseLocation string,
) (*ShortLinkLookupPayload, error) {
	if r == nil {
		return nil, errors.New("missing request")
	}
	payload := &ShortLinkLookupPayload{
		ShortLinkID:        shortLinkID,
		ShortLinkFound:     shortLink != nil,
		ShortLinkExpired:   shortLink.Expired(),
		TargetURL:          shortLink.LinkURL,
		RequestHost:        r.Host,
		RequestHeaders:     make(map[string]string),
		RequestedAt:        time.Now().Format(time.RFC3339),
		ResponseStatusCode: responseStatusCode,
		ResponseLocation:   responseLocation,
	}

	for _, key := range pb.trustedHeaders {
		value := r.Header.Get(key)
		// basically emulating "json:omitempty"
		if value != "" {
			payload.RequestHeaders[key] = value
		}
	}

	return payload, nil
}

var DefaultAWSTrustedHeaders = []string{
	"User-Agent",
	"X-Forwarded-For",
	"CloudFront-Viewer-Country",
	"CloudFront-Viewer-City",
	"CloudFront-Is-Desktop-Viewer",
	"CloudFront-Is-Mobile-Viewer",
	"CloudFront-Is-SmartTV-Viewer",
	"CloudFront-Is-Tablet-Viewer",
}

var DefaultCloudflareTrustedHeaders = []string{
	"User-Agent",
	"CF-Connecting-IP",
	"CF-IP-Country",
}
