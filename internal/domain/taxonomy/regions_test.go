package taxonomy

import "testing"

func TestEveryCountryHasExactlyOneRegion(t *testing.T) {
	seen := make(map[string]int, len(CountryRegion))
	for _, c := range Countries() {
		if c.Code == CountryOther.Code {
			// "Other" is a fallback bucket, not a place on the map.
			continue
		}
		if c.Code == "AQ" {
			// Antarctica is selectable but has no region preset or map shape.
			continue
		}
		region, ok := CountryRegion[c.Code]
		if !ok {
			t.Errorf("country %s (%s) has no region", c.Code, c.Name)
			continue
		}
		if region == "" || region == RegionWorld {
			t.Errorf("country %s has invalid region %q", c.Code, region)
		}
		seen[c.Code]++
	}
	for code, n := range seen {
		if n != 1 {
			t.Errorf("country %s is assigned to %d regions, want 1", code, n)
		}
	}
}

func TestRegionOf(t *testing.T) {
	tests := map[string]Region{
		"US": RegionNorthAmerica,
		"BR": RegionSouthAmerica,
		"GB": RegionEurope,
		"NG": RegionAfrica,
		"JP": RegionAsia,
		"AU": RegionOceania,
	}
	for code, want := range tests {
		if got := RegionOf(code); got != want {
			t.Errorf("RegionOf(%q) = %q, want %q", code, got, want)
		}
	}
	if RegionOf("ZZ") != "" {
		t.Error("RegionOf(ZZ) should be empty for an unknown code")
	}
}

func TestRegionViewBoxStaysInsideWorld(t *testing.T) {
	for _, r := range AllRegions() {
		vb := RegionViewBox(r)
		if vb[2] <= 0 || vb[3] <= 0 {
			t.Errorf("region %q has non-positive viewBox size: %v", r, vb)
			continue
		}
		if vb[0] < 0 || vb[1] < 0 {
			t.Errorf("region %q has negative viewBox origin: %v", r, vb)
		}
		if vb[0]+vb[2] > 880.001 || vb[1]+vb[3] > 440.001 {
			t.Errorf("region %q extends past the world: %v", r, vb)
		}
		// Presets keep the 2:1 world aspect so the panel does not letterbox.
		if got, want := vb[2]/vb[3], 2.0; got < want*0.9 || got > want*1.1 {
			t.Errorf("region %q aspect = %.2f, want about 2", r, got)
		}
	}
}
