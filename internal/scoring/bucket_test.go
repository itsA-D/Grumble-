package scoring

import "testing"

func TestBucketFor(t *testing.T) {
	tests := []struct {
		name  string
		count int
		max   int
		want  Bucket
	}{
		{name: "zero count is no data", count: 0, max: 100, want: BucketNone},
		{name: "zero max is no data", count: 5, max: 0, want: BucketNone},
		{name: "negative count is no data", count: -3, max: 100, want: BucketNone},
		{name: "single country is maximum", count: 1, max: 1, want: Bucket5},
		{name: "top country is maximum", count: 100, max: 100, want: Bucket5},
		{name: "smallest positive is low", count: 1, max: 100, want: Bucket1},
		{name: "log scale lifts small counts", count: 3, max: 100, want: Bucket2},
		{name: "midpoint stays mid", count: 10, max: 100, want: Bucket3},
		{name: "near top is high", count: 50, max: 100, want: Bucket4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BucketFor(tt.count, tt.max); got != tt.want {
				t.Fatalf("BucketFor(%d, %d) = %d, want %d", tt.count, tt.max, got, tt.want)
			}
		})
	}
}

func TestBucketForIsMonotonic(t *testing.T) {
	prev := BucketNone
	for count := 0; count <= 100; count++ {
		b := BucketFor(count, 100)
		if b < prev {
			t.Fatalf("bucket decreased at count=%d: %d after %d", count, b, prev)
		}
		prev = b
	}
}
