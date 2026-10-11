package web

import (
	"net/http"
	"strconv"

	"grumble/internal/service/feed"
	"grumble/internal/transport/web/seo"
	"grumble/internal/transport/web/views"
)

// feedPageSize is how many pains one feed page shows.
const feedPageSize = 20

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
	tab := r.URL.Query().Get("tab")
	feedTab, ok := feedTabs[tab]
	if !ok {
		tab, feedTab = "trending", feed.Trending
	}
	pageNum := parsePage(r.URL.Query().Get("page"))

	pains, err := s.deps.Feed.List(r.Context(), feed.Query{
		Tab:    feedTab,
		Limit:  feedPageSize,
		Offset: (pageNum - 1) * feedPageSize,
	})
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	cards := s.cardsFromPains(pains)
	s.render(w, r, http.StatusOK, views.Feed(views.FeedPage{
		Page:      s.page(r.Context(), seo.Feed(s.deps.ProductName, tab)),
		Tab:       tab,
		TabLabel:  feedTabLabels[tab],
		Cards:     cards,
		PageNum:   pageNum,
		HasPrev:   pageNum > 1,
		HasNext:   len(cards) == feedPageSize,
		PrevPage:  pageNum - 1,
		NextPage:  pageNum + 1,
		EmptyFeed: len(cards) == 0,
	}))
}

func parsePage(raw string) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}
