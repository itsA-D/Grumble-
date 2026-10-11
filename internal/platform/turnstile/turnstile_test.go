package turnstile

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDisabledAlwaysPasses(t *testing.T) {
	c := New(false, "", nil)
	if err := c.Verify(context.Background(), ""); err != nil {
		t.Fatalf("disabled verifier must pass, got %v", err)
	}
	// Disabled ignores token content, even session-looking values.
	if err := c.Verify(context.Background(), "anything"); err != nil {
		t.Fatalf("disabled verifier must pass any token, got %v", err)
	}
}

func TestNoopAlwaysPasses(t *testing.T) {
	if err := (Noop{}).Verify(context.Background(), ""); err != nil {
		t.Fatalf("Noop must pass, got %v", err)
	}
}

func TestClientRejectsEmptyToken(t *testing.T) {
	c := New(true, "secret", nil)
	if err := c.Verify(context.Background(), ""); !strings.HasSuffix(err.Error(), ErrChallengeFailed.Error()) {
		t.Fatalf("empty token must fail, got %v", err)
	}
}

func TestClientVerifiesAgainstEndpoint(t *testing.T) {
	var gotSecret, gotResponse string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %q", ct)
		}
		body, _ := io.ReadAll(r.Body)
		vals := parseForm(body)
		gotSecret, gotResponse = vals["secret"], vals["response"]
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	c := New(true, "s3cr3t", srv.Client())
	c.endpoint = srv.URL

	if err := c.Verify(context.Background(), "token-123"); err != nil {
		t.Fatalf("valid challenge rejected: %v", err)
	}
	if gotSecret != "s3cr3t" || gotResponse != "token-123" {
		t.Errorf("siteverify payload secret=%q response=%q", gotSecret, gotResponse)
	}
}

func TestClientRejectsUnsuccessfulResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer srv.Close()

	c := New(true, "s3cr3t", srv.Client())
	c.endpoint = srv.URL
	if err := c.Verify(context.Background(), "bad"); !errors.Is(err, ErrChallengeFailed) {
		t.Fatalf("error = %v, want ErrChallengeFailed", err)
	}
}

func TestClientSurfacesUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := New(true, "s3cr3t", srv.Client())
	c.endpoint = srv.URL
	if err := c.Verify(context.Background(), "token"); !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected an error mentioning the status, got %v", err)
	}
}

func TestClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Millisecond}
	c := New(true, "s3cr3t", client)
	c.endpoint = srv.URL
	if err := c.Verify(context.Background(), "token"); err == nil {
		t.Fatal("expected a timeout error")
	}
}

func parseForm(b []byte) map[string]string {
	out := make(map[string]string)
	for _, kv := range strings.Split(string(b), "&") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}
