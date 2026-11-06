package scoring

import "math"

// Bucket is the intensity bucket for the choropleth map: 0 (no data) to 5
// (maximum intensity in the current filter).
type Bucket int

// Intensity buckets, from no data to the maximum in the current filter.
const (
	BucketNone Bucket = iota // no published pains
	Bucket1
	Bucket2
	Bucket3
	Bucket4
	Bucket5
)

// BucketFor returns the intensity bucket for a country's pain count, using a
// log-scaled 5-bucket scheme across the filtered maximum. The maximum is the
// highest pain_count among the countries in the current view (category filter).
// If maxCount <= 0, all countries get BucketNone.
func BucketFor(count, maxCount int) Bucket {
	if count <= 0 || maxCount <= 0 {
		return BucketNone
	}
	// Log-scaled buckets: 1-5 map to roughly 20%, 38%, 56%, 76%, 100% of max.
	// We use log1p(count) / log1p(maxCount) to spread small values and compress
	// large ones, then scale to 5 buckets with round-half-up.
	ratio := math.Log1p(float64(count)) / math.Log1p(float64(maxCount))
	b := int(math.Round(ratio * 5))
	if b < 1 {
		return Bucket1
	}
	if b > 5 {
		return Bucket5
	}
	return Bucket(b)
}

// BucketColor returns the CSS color-mix value for a bucket (FR-MAP-1).
// Bucket 0 = rgb(255 255 255 / .06)
// Buckets 1-5 = color-mix(in srgb, var(--sig) X%, transparent)
// where X = 20, 38, 56, 76, 100.
func BucketColor(b Bucket) string {
	switch b {
	case BucketNone:
		return "rgb(255 255 255 / 0.06)"
	case Bucket1:
		return "color-mix(in srgb, var(--sig) 20%, transparent)"
	case Bucket2:
		return "color-mix(in srgb, var(--sig) 38%, transparent)"
	case Bucket3:
		return "color-mix(in srgb, var(--sig) 56%, transparent)"
	case Bucket4:
		return "color-mix(in srgb, var(--sig) 76%, transparent)"
	case Bucket5:
		return "color-mix(in srgb, var(--sig) 100%, transparent)"
	}
	return "rgb(255 255 255 / 0.06)"
}
