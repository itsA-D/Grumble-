package web

import (
	"net/http"
	"strings"

	"grumble/internal/domain/taxonomy"
	"grumble/internal/service/feed"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

// feedPageSize is how many pains one feed page shows.
const feedPageSize = 20

// maxSearchRunes bounds the search term the feed handler forwards.
const maxSearchRunes = 100

var feedTabs = map[string]feed.Tab{
	"trending":   feed.Trending,
	"latest":     feed.Latest,
	"most_voted": feed.MostVoted,
}

var feedTabLabels = map[string]string{
	"trending":   "Trending",
	"latest":     "Latest",
	"most_voted": "Most voted",
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	tab, feedTab := resolveFeedTab(q.Get("tab"))
	category := resolveFeedCategory(q.Get("category"))
	search := truncateRunes(strings.TrimSpace(q.Get("q")), maxSearchRunes)
	cursor := q.Get("cursor")

	page, err := s.deps.Feed.List(r.Context(), feed.Query{
		Tab:      feedTab,
		Category: category,
		Search:   search,
		Cursor:   cursor,
		Limit:    feedPageSize,
	})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	cards := s.cardsFromPains(page.Pains)
	s.render(w, r, http.StatusOK, views.Feed(views.FeedPage{
		Page:           s.page(r.Context(), seo.Feed(s.deps.ProductName, tab)),
		Tab:            tab,
		TabLabel:       feedTabLabels[tab],
		Categories:     taxonomy.Categories(),
		ActiveCategory: category,
		Search:         search,
		Cards:          cards,
		NextCursor:     page.NextCursor,
		HasNext:        page.HasNext,
		HasCursor:      cursor != "",
		EmptyFeed:      len(cards) == 0,
	}))
}

// resolveFeedTab maps a tab query value to its tab and canonical name, defaulting
// to Trending.
func resolveFeedTab(raw string) (string, feed.Tab) {
	if tab, ok := feedTabs[raw]; ok {
		return raw, tab
	}
	return "trending", feed.Trending
}

// resolveFeedCategory keeps only a known category slug; anything else is treated
// as "no filter" so the feed still renders.
func resolveFeedCategory(raw string) string {
	if _, ok := taxonomy.CategoryBySlug(raw); ok {
		return raw
	}
	return ""
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}
