package postgres

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"grumble/internal/domain/pain"
	"grumble/internal/domain/taxonomy"
	"grumble/internal/store/postgres/queries"
)

// The published_pains view yields one shared row shape (query.PublishedPain),
// so public reads funnel through one mapper.

func painFromPublished(r queries.PublishedPain) pain.Pain {
	return pain.Pain{
		ID:          r.ID,
		Slug:        r.Slug,
		Body:        r.Body,
		Category:    category(r.Category),
		Country:     country(r.Country),
		Details:     detailsFrom(r.Frequency, r.Disruption, r.WouldPay, r.MonthlyAmount, r.Tried),
		Status:      pain.Status(r.Status),
		VoteCount:   int(r.VoteCount),
		Urgency:     int(r.Urgency),
		Seeded:      r.Seeded,
		CreatedAt:   timeOf(r.CreatedAt),
		PublishedAt: optionalTime(r.PublishedAt),
	}
}

func painFromModel(r queries.Pain) pain.Pain {
	return pain.Pain{
		ID:          r.ID,
		Slug:        r.Slug,
		Body:        r.Body,
		Category:    category(r.Category),
		Country:     country(r.Country),
		Details:     detailsFrom(r.Frequency, r.Disruption, r.WouldPay, r.MonthlyAmount, r.Tried),
		Status:      pain.Status(r.Status),
		VoteCount:   int(r.VoteCount),
		Urgency:     int(r.Urgency),
		Seeded:      r.Seeded,
		CreatedAt:   timeOf(r.CreatedAt),
		PublishedAt: optionalTime(r.PublishedAt),
	}
}

func painFromTrendingRow(r queries.ListPublishedTrendingCandidatesRow) pain.Pain {
	return pain.Pain{
		ID:          r.ID,
		Slug:        r.Slug,
		Body:        r.Body,
		Category:    category(r.Category),
		Country:     country(r.Country),
		Details:     detailsFrom(r.Frequency, r.Disruption, r.WouldPay, r.MonthlyAmount, r.Tried),
		Status:      pain.Status(r.Status),
		VoteCount:   int(r.VoteCount),
		Urgency:     int(r.Urgency),
		Seeded:      r.Seeded,
		CreatedAt:   timeOf(r.CreatedAt),
		PublishedAt: optionalTime(r.PublishedAt),
	}
}

func mapPublished(rows []queries.PublishedPain, err error) ([]pain.Pain, error) {
	if err != nil {
		return nil, err
	}
	out := make([]pain.Pain, 0, len(rows))
	for _, row := range rows {
		out = append(out, painFromPublished(row))
	}
	return out, nil
}

func detailsFrom(freq, disruption *string, wouldPay bool, amount *int32, tried *string) pain.Details {
	return pain.Details{
		Frequency:     pain.Frequency(deref(freq)),
		Disruption:    pain.Disruption(deref(disruption)),
		WouldPay:      wouldPay,
		MonthlyAmount: int(derefInt32(amount)),
		Tried:         deref(tried),
	}
}

// category resolves a stored slug back to the catalog entry, tolerating a slug
// that is no longer in the catalog (it still renders as plain text).
func category(slug string) taxonomy.Category {
	if c, ok := taxonomy.CategoryBySlug(slug); ok {
		return c
	}
	return taxonomy.Category{Slug: slug, Name: slug}
}

func country(code string) taxonomy.Country {
	if c, ok := taxonomy.CountryByCode(code); ok {
		return c
	}
	return taxonomy.Country{Code: code, Name: code}
}

func timeOf(ts pgtype.Timestamptz) time.Time {
	if ts.Valid {
		return ts.Time
	}
	return time.Time{}
}

func optionalTime(ts pgtype.Timestamptz) *time.Time {
	if ts.Valid {
		t := ts.Time
		return &t
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalInt32(i int) *int32 {
	if i <= 0 {
		return nil
	}
	v := int32(i)
	return &v
}
