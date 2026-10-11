// Package identity derives a stable, privacy-preserving client identity
// (PRD NFR-PRIV-1): a signed anonymous id for voting, and a monthly-rotated
// salted hash of the client address used for moderation and rate limiting.
// Raw IP addresses are never stored or logged anywhere in the app.
package identity

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/netip"
	"strings"
	"time"
)

// Cookie and token limits.
const (
	// AnonIDVersion prefixes signed anonymous ids.
	AnonIDVersion = "v1"
	// AnonIDPrefix separates the HMAC domain so ids signed elsewhere cannot collide.
	anonIDPrefix = "painboard.anonid."
	// AnonIDMaxAge bounds how long the anonymous id cookie is accepted.
	AnonIDMaxAge = 90 * 24 * time.Hour
	// HashLen is the hex length of a stored source hash.
	HashLen = 24
)

// Identity signs anonymous ids and hashes client addresses. Salts come from
// configuration; the monthly bucket provides the required rotation (NFR-PRIV-1).
type Identity struct {
	idSecret     []byte
	saltCurrent  string
	saltPrevious string
}

// New builds an Identity. idSecret signs anon ids; saltCurrent salts source
// hashes. saltPrevious, when set, keeps matching repeat visitors during salt
// rotation.
func New(idSecret, saltCurrent, saltPrevious string) *Identity {
	return &Identity{
		idSecret:     []byte(idSecret),
		saltCurrent:  saltCurrent,
		saltPrevious: saltPrevious,
	}
}

// SignAnonID returns a tamper-evident token carrying the raw anon id.
func (i *Identity) SignAnonID(id string) string {
	sig := i.mac(anonIDPrefix + id)
	return AnonIDVersion + "." + id + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// VerifyAnonID accepts a signed token and returns the raw anon id. It rejects
// tampered, malformed and foreign ids.
func (i *Identity) VerifyAnonID(token string) (string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != AnonIDVersion {
		return "", false
	}
	id := parts[1]
	want := base64.RawURLEncoding.EncodeToString(i.mac(anonIDPrefix + id))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return "", false
	}
	return id, true
}

func (i *Identity) mac(msg string) []byte {
	m := hmac.New(sha256.New, i.idSecret)
	m.Write([]byte(msg))
	return m.Sum(nil)
}

// SourceHash returns the hash stored with a submission: the current salt in the
// current monthly bucket. See SourceHashes for the candidates to match against.
func (i *Identity) SourceHash(ip string, at time.Time) string {
	return sourceHash(i.saltCurrent, ip, at)
}

// SourceHashes returns every reasonable hash value for a client address at time
// at: the current and previous monthly buckets under the current salt, plus the
// same under the previous salt when one is configured. Use this list when
// matching stored submissions so repeat visitors keep matching across the
// monthly rotation.
func (i *Identity) SourceHashes(ip string, at time.Time) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, 4)
	add := func(salt string, months time.Time) {
		h := sourceHash(salt, ip, months)
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	add(i.saltCurrent, at)
	add(i.saltCurrent, at.AddDate(0, -1, 0))
	if i.saltPrevious != "" {
		add(i.saltPrevious, at)
		add(i.saltPrevious, at.AddDate(0, -1, 0))
	}
	return out
}

// sourceHash is sha256(salt + ":" + yyyy-mm + ":" + normalizedIP), truncated.
// The bucket rotates monthly so a compromise of one bucket does not leak the
// identity of the same visitor in later months.
func sourceHash(salt, ip string, at time.Time) string {
	bucket := at.UTC().Format("2006-01")
	sum := sha256.Sum256([]byte(salt + ":" + bucket + ":" + normalizeIP(ip)))
	return hex.EncodeToString(sum[:])[:HashLen]
}

// normalizeIP returns a canonical form so one client has one hash. Unparsable
// strings are lowered and trimmed rather than rejected: the middleware derives
// the value from trusted headers, so a hash is still produced but it never
// matches a canonical address.
func normalizeIP(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if addr, err := netip.ParseAddr(s); err == nil {
		return addr.String()
	}
	return s
}
