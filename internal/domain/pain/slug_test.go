package pain

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"I hate when my coffee is cold", "i-hate-when-my-coffee-is-cold"},
		{"  Leading and trailing  ", "leading-and-trailing"},
		{"Punct-uation!!! and (parens)", "punct-uation-and-parens"},
		{"Café déjà vu ☕", "cafe-deja-vu"},
		{"Multiple   spaces", "multiple-spaces"},
		{"", ""},
		{"!!!", ""},
		{"Slug_under/score", "slug-under-score"},
	}
	for _, tc := range cases {
		if got := Slugify(tc.in); got != tc.want {
			t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSlugifyBounds(t *testing.T) {
	long := ""
	for i := 0; i < 40; i++ {
		long += "word "
	}
	got := Slugify(long)
	if len(got) > MaxSlugLen {
		t.Errorf("slug too long: %d > %d (%q)", len(got), MaxSlugLen, got)
	}
}
