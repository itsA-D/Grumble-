package identity

import (
	"strings"
	"testing"
	"time"
)

func TestAnonIDRoundTrip(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	raw := "e3b0c442-98fc-11ef-8f8c-0242ac120002"
	token := id.SignAnonID(raw)

	if got, ok := id.VerifyAnonID(token); !ok || got != raw {
		t.Fatalf("VerifyAnonID(%q) = %q, %v; want %q, true", token, got, ok, raw)
	}
}

func TestAnonIDRejectsTampering(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	token := id.SignAnonID("raw-id")

	cases := []string{
		"",
		"v1.raw-id",
		"v1.raw-id.dG9rZW4=",
		"v1.other-id." + strings.Split(token, ".")[2],
		"v2.raw-id." + strings.Split(token, ".")[2],
	}
	for _, tc := range cases {
		if got, ok := id.VerifyAnonID(tc); ok {
			t.Errorf("VerifyAnonID(%q) accepted as %q", tc, got)
		}
	}
}

func TestAnonIDKeysAreSeparated(t *testing.T) {
	a := New("secret-for-a-1234567890", "salt-a-1234567890", "")
	b := New("secret-for-b-1234567890", "salt-b-1234567890", "")
	if _, ok := b.VerifyAnonID(a.SignAnonID("raw")); ok {
		t.Error("token signed with one secret verified under another")
	}
}

func TestCSRFTokenRoundTripAndBinding(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	token := id.CSRFToken("anon-1")

	if !id.VerifyCSRF("anon-1", token) {
		t.Fatal("VerifyCSRF rejected its own token")
	}
	if id.VerifyCSRF("anon-2", token) {
		t.Error("token issued for anon-1 verified for anon-2")
	}
	if id.VerifyCSRF("anon-1", token+"x") {
		t.Error("tampered token verified")
	}
	if id.VerifyCSRF("", token) || id.VerifyCSRF("anon-1", "") {
		t.Error("empty identity or token must not verify")
	}
	if id.CSRFToken("anon-1") == id.SignAnonID("anon-1") {
		t.Error("CSRF and anon-id signatures must be domain-separated")
	}
}

func TestSourceHashStableWithinBucket(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	at := time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)
	later := time.Date(2026, time.March, 25, 12, 0, 0, 0, time.UTC)
	if id.SourceHash("192.0.2.10", at) != id.SourceHash("192.0.2.10", later) {
		t.Error("hash changed within the same month bucket")
	}
}

func TestSourceHashRotatesMonthly(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	march := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	april := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	if id.SourceHash("192.0.2.10", march) == id.SourceHash("192.0.2.10", april) {
		t.Error("hash must change across months")
	}
}

func TestSourceHashesCandidateSet(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "salt-b-1234567890")
	at := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	hashes := id.SourceHashes("192.0.2.10", at)

	// Current bucket current salt, previous bucket current salt, plus the same
	// under the previous salt: four distinct values.
	if len(hashes) != 4 {
		t.Fatalf("SourceHashes() = %d entries, want 4: %q", len(hashes), hashes)
	}

	// The previous salt alone must not reproduce the current salt's hash.
	if id.SourceHash("192.0.2.10", at) == New("s", "salt-b-1234567890", "").SourceHash("192.0.2.10", at) {
		t.Error("different salts must hash differently")
	}

	// Without a previous salt only two candidates remain.
	short := New("super-secret-1234567890", "salt-a-1234567890", "")
	hs := short.SourceHashes("192.0.2.10", at)
	if len(hs) != 2 {
		t.Fatalf("without previous salt got %d candidates, want 2", len(hs))
	}
}

func TestSourceHashCanonicalisesIP(t *testing.T) {
	id := New("super-secret-1234567890", "salt-a-1234567890", "")
	at := time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC)
	// Same address in two textual forms must hash identically.
	// IPv4-mapped IPv6 resolves to the same canonical IPv4 address.
	if id.SourceHash("2001:db8::1", at) != id.SourceHash("2001:0db8:0000::1", at) {
		t.Error("IPv6 forms must canonicalise to one hash")
	}
	if id.SourceHash("192.0.2.1", at) == id.SourceHash("2001:db8::1", at) {
		t.Error("distinct addresses must hash differently")
	}
}
