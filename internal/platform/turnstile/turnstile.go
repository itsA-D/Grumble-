// Package turnstile verifies Cloudflare Turnstile challenge tokens
// (PRD FR-SUB-2). When Turnstile is disabled the Noop verifier is used, which
// keeps the rest of the code unchanged in dev and locally.
package turnstile

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DefaultEndpoint is the siteverify API. Overridable in New for tests.
const DefaultEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// ErrChallengeFailed marks an invalid, missing or failed challenge.
var ErrChallengeFailed = errors.New("turnstile: challenge failed")

// Verifier reports whether a submitted challenge token is valid.
type Verifier interface {
	// Verify returns nil when the token passes (or when verification is
	// disabled). Any other result is ErrChallengeFailed.
	Verify(ctx context.Context, token string) error
}

// Client implements Verifier against the Cloudflare API.
type Client struct {
	enabled  bool
	secret   string
	endpoint string
	httpc    *http.Client
}

// New builds a verifier from config. With enabled=false the client always
// passes challenges (dev mode). hc must be non-nil.
func New(enabled bool, secret string, hc *http.Client) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &Client{
		enabled:  enabled,
		secret:   secret,
		endpoint: DefaultEndpoint,
		httpc:    hc,
	}
}

// Verify implements Verifier.
func (c *Client) Verify(ctx context.Context, token string) error {
	if !c.enabled {
		return nil
	}
	if strings.TrimSpace(token) == "" {
		return ErrChallengeFailed
	}

	form := url.Values{
		"secret":   {c.secret},
		"response": {token},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint,
		bytes.NewBufferString(form.Encode()))
	if err != nil {
		return fmt.Errorf("turnstile: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile: call siteverify: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turnstile: siteverify returned %d: %w", resp.StatusCode, ErrChallengeFailed)
	}

	var out struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("turnstile: decode response: %w", err)
	}
	if !out.Success {
		return ErrChallengeFailed
	}
	return nil
}

// Noop always accepts; used when an explicit no-op is wanted over Client.
type Noop struct{}

// Verify implements Verifier and always succeeds.
func (Noop) Verify(context.Context, string) error { return nil }
