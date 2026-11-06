package ui

import "testing"

func TestFormatCount(t *testing.T) {
	cases := map[int]string{
		-1:     "0",
		0:      "0",
		7:      "7",
		999:    "999",
		1000:   "1,000",
		1240:   "1,240",
		8421:   "8,421",
		100000: "100,000",
	}
	for in, want := range cases {
		if got := FormatCount(in); got != want {
			t.Fatalf("FormatCount(%d) = %q, want %q", in, got, want)
		}
	}
}
