package pain

import (
	"time"

	"github.com/google/uuid"
)

// FeedOrder selects the SQL ordering of a feed page (FR-FEED-1).
type FeedOrder int

const (
	// FeedLatest orders newest first.
	FeedLatest FeedOrder = iota
	// FeedTopVotes orders by vote count first.
	FeedTopVotes
)

// FeedCursor is a keyset position: the sort key of the last row on the previous
// page. Which fields matter depends on the order. The zero value is not a valid
// position; callers that want the first page pass the service's start cursor.
type FeedCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
	Votes     int
}

// FeedFilter is one published-only feed page request. Category and Search, when
// set, narrow the page; Search is matched as a case-insensitive plain substring
// (never as a wildcard pattern). Limit is the page size plus one, so the caller
// can tell whether a further page exists.
type FeedFilter struct {
	Order    FeedOrder
	Category string
	Search   string
	After    FeedCursor
	Limit    int
}
