// Package idgen generates time-ordered identifiers (UUIDv7) so rows sort by
// creation time and indexes stay cache-friendly.
package idgen

import (
	"fmt"

	"github.com/google/uuid"
)

// New returns a new UUIDv7.
func New() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate uuidv7: %w", err)
	}
	return id, nil
}

// MustNew returns a new UUIDv7 and panics only if the system entropy source
// fails, which is unrecoverable. Use it where an error return is impractical.
func MustNew() uuid.UUID {
	id, err := New()
	if err != nil {
		panic(err)
	}
	return id
}

// String returns a new UUIDv7 in its canonical string form.
func String() (string, error) {
	id, err := New()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
