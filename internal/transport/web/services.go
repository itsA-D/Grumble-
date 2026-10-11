package web

import (
	"context"

	"github.com/google/uuid"

	"github.com/OWNER/painboard/internal/domain/pain"
	"github.com/OWNER/painboard/internal/service/feed"
	"github.com/OWNER/painboard/internal/service/hubs"
	"github.com/OWNER/painboard/internal/service/pains"
	"github.com/OWNER/painboard/internal/service/voting"
)

// The transport declares the service surface it needs; the composition root
// passes the concrete services, which satisfy these (ARCHITECTURE.md section 9).

// FeedService returns a page of the explore feed (FR-FEED-1/2).
type FeedService interface {
	List(ctx context.Context, q feed.Query) ([]pain.Pain, error)
}

// PainsService submits a pain, reads one detail page and lists recent pains for
// the sitemap.
type PainsService interface {
	Submit(ctx context.Context, req pains.SubmitRequest) (pains.SubmitResult, error)
	Get(ctx context.Context, slug string) (pains.Detail, error)
	RecentForSitemap(ctx context.Context, limit int) ([]pain.Pain, error)
}

// VotingService toggles a vote and reports the voter's current state.
type VotingService interface {
	Set(ctx context.Context, req voting.Request) (voting.Result, error)
	Voted(ctx context.Context, painID uuid.UUID, voterKey string) (bool, error)
}

// HubsService builds category and country landing pages (FR-HUB-1).
type HubsService interface {
	Get(ctx context.Context, q hubs.Query) (hubs.Hub, error)
}

// IDSigner signs and verifies the anonymous-id cookie (NFR-PRIV-1) and derives
// the per-identity CSRF token (NFR-SEC-2).
type IDSigner interface {
	SignAnonID(id string) string
	VerifyAnonID(token string) (string, bool)
	CSRFToken(anonID string) string
	VerifyCSRF(anonID, token string) bool
}
