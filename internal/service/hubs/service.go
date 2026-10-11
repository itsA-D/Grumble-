// Package hubs builds category and country landing pages: a ranked list, an
// original intro and FAQ, and the indexability rule (PRD FR-HUB-1, FR-SEO-1,
// FR-SEO-3). Copy here is original product text; it never includes a count of
// zero.
package hubs

import (
	"context"
	"fmt"

	"github.com/OWNER/painboard/internal/domain"
	"github.com/OWNER/painboard/internal/domain/pain"
	"github.com/OWNER/painboard/internal/domain/taxonomy"
)

// MinPublishedForIndex is the published-pain floor below which a hub is
// noindexed (FR-SEO-3).
const MinPublishedForIndex = 5

// Page-size bounds.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// Kind distinguishes the two hub axes.
type Kind string

const (
	// Category hubs group pains by category slug.
	Category Kind = "category"
	// Country hubs group pains by ISO code.
	Country Kind = "country"
)

// Store is the persistence the service needs.
type Store interface {
	CountByCategory(ctx context.Context, category string) (int, error)
	CountByCountry(ctx context.Context, country string) (int, error)
	ListByCategory(ctx context.Context, category string, limit, offset int) ([]pain.Pain, error)
	ListByCountry(ctx context.Context, country string, limit, offset int) ([]pain.Pain, error)
}

// Service reads hubs.
type Service struct {
	store Store
}

// New builds the hubs service.
func New(store Store) *Service {
	return &Service{store: store}
}

// Query identifies one hub page.
type Query struct {
	Kind   Kind
	Slug   string
	Limit  int
	Offset int
}

// QA is one FAQ entry.
type QA struct {
	Question string
	Answer   string
}

// Hub is a rendered landing page.
type Hub struct {
	Kind        Kind
	Slug        string
	Title       string
	Description string
	Intro       string
	FAQ         []QA
	Pains       []pain.Pain
	Count       int
	// Indexable is false while the hub has fewer than MinPublishedForIndex
	// published pains (FR-SEO-3).
	Indexable bool
}

// Get resolves a hub, its ranked pains and its copy. An unknown axis value or
// slug returns domain.ErrNotFound.
func (s *Service) Get(ctx context.Context, q Query) (Hub, error) {
	limit, offset := normalize(q)
	switch q.Kind {
	case Category:
		return s.categoryHub(ctx, q.Slug, limit, offset)
	case Country:
		return s.countryHub(ctx, q.Slug, limit, offset)
	default:
		return Hub{}, domain.ErrNotFound
	}
}

func (s *Service) categoryHub(ctx context.Context, slug string, limit, offset int) (Hub, error) {
	cat, ok := taxonomy.CategoryBySlug(slug)
	if !ok {
		return Hub{}, domain.ErrNotFound
	}
	count, err := s.store.CountByCategory(ctx, cat.Slug)
	if err != nil {
		return Hub{}, err
	}
	pains, err := s.store.ListByCategory(ctx, cat.Slug, limit, offset)
	if err != nil {
		return Hub{}, err
	}
	subject := "in " + cat.Name
	return Hub{
		Kind:        Category,
		Slug:        cat.Slug,
		Title:       "Frustrations about " + cat.Name,
		Description: description(count, "frustrations about "+cat.Name),
		Intro:       intro(subject),
		FAQ:         faq(subject, "Post it on the main board and choose this category when you file it."),
		Pains:       pains,
		Count:       count,
		Indexable:   count >= MinPublishedForIndex,
	}, nil
}

func (s *Service) countryHub(ctx context.Context, code string, limit, offset int) (Hub, error) {
	country, ok := taxonomy.CountryByCode(code)
	if !ok {
		return Hub{}, domain.ErrNotFound
	}
	count, err := s.store.CountByCountry(ctx, country.Code)
	if err != nil {
		return Hub{}, err
	}
	pains, err := s.store.ListByCountry(ctx, country.Code, limit, offset)
	if err != nil {
		return Hub{}, err
	}
	subject := "in " + country.Name
	return Hub{
		Kind:        Country,
		Slug:        country.Code,
		Title:       "Frustrations in " + country.Name,
		Description: description(count, "frustrations in "+country.Name),
		Intro:       intro(subject),
		FAQ:         faq(subject, "Post it on the main board and choose "+country.Name+" as the country when you file it."),
		Pains:       pains,
		Count:       count,
		Indexable:   count >= MinPublishedForIndex,
	}, nil
}

// description builds the meta description. It never states a count of zero
// (FR-SEO-1); a hub with no published pains gets count-free copy instead.
func description(count int, subject string) string {
	if count <= 0 {
		return "Browse " + subject + ", ranked by how urgent the community finds them."
	}
	return fmt.Sprintf("Browse %d %s, ranked by how urgent the community finds them.", count, subject)
}

// intro is the hub's original lead paragraph.
func intro(subject string) string {
	return "This is a live list of the frustrations people are posting " + subject +
		". Each one is ranked by how much it disrupts life and how many people back it, so the " +
		"problems that hit hardest rise to the top. Vote for anything you recognise to push it higher, " +
		"and add your own when you cannot find it."
}

// faq is the hub's original FAQ, adapted to the axis. addHint is the full
// sentence telling a reader how to file a pain on this axis. The answers state
// how ranking works without inventing counts.
func faq(subject, addHint string) []QA {
	return []QA{
		{
			Question: "What is this page?",
			Answer:   "A running list of frustrations people have posted " + subject + ", ordered so the most urgent ones appear first.",
		},
		{
			Question: "How is the order decided?",
			Answer: "Each frustration carries an urgency score. It combines how often the problem happens, how much it " +
				"interferes with daily life, whether the person would pay to fix it, and how many votes it has collected.",
		},
		{
			Question: "How do I add my own?",
			Answer:   addHint,
		},
	}
}

func normalize(q Query) (limit, offset int) {
	limit = q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	offset = q.Offset
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
