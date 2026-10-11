package identity

import (
	"crypto/hmac"
	"encoding/base64"
)

// csrfPrefix separates the CSRF token domain from the anon-id domain so a
// signature from one can never be replayed as the other.
const csrfPrefix = "grumble.csrf."

// CSRFToken derives a per-identity synchronizer token. It is bound to the
// anonymous id carried by the signed, httpOnly cookie, so a cross-site page
// cannot read the id (let alone forge the token) and cannot include it in a
// request. It is compared in constant time by VerifyCSRF.
func (i *Identity) CSRFToken(anonID string) string {
	return base64.RawURLEncoding.EncodeToString(i.mac(csrfPrefix + anonID))
}

// VerifyCSRF reports whether token is the current token for anonID. Empty
// inputs never verify.
func (i *Identity) VerifyCSRF(anonID, token string) bool {
	if anonID == "" || token == "" {
		return false
	}
	return hmac.Equal([]byte(i.CSRFToken(anonID)), []byte(token))
}
