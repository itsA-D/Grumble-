package taxonomy

// Region is a world region used for map preset views.
type Region string

// World regions. RegionWorld is the neutral default preset.
const (
	RegionWorld        Region = "world"
	RegionNorthAmerica Region = "north_america"
	RegionSouthAmerica Region = "south_america"
	RegionEurope       Region = "europe"
	RegionAfrica       Region = "africa"
	RegionAsia         Region = "asia"
	RegionOceania      Region = "oceania"
)

// AllRegions returns the regions in display order (World first for the
// "all countries" chip).
func AllRegions() []Region {
	return []Region{
		RegionWorld,
		RegionNorthAmerica,
		RegionSouthAmerica,
		RegionEurope,
		RegionAfrica,
		RegionAsia,
		RegionOceania,
	}
}

// RegionName returns the human-readable name for a region.
func RegionName(r Region) string {
	switch r {
	case RegionWorld:
		return "World"
	case RegionNorthAmerica:
		return "N. America"
	case RegionSouthAmerica:
		return "S. America"
	case RegionEurope:
		return "Europe"
	case RegionAfrica:
		return "Africa"
	case RegionAsia:
		return "Asia"
	case RegionOceania:
		return "Oceania"
	}
	return ""
}

// CountryRegion maps an ISO 3166-1 alpha-2 code to its world region.
// Every country in the taxonomy has exactly one region.
var CountryRegion = map[string]Region{
	// North America
	"US": RegionNorthAmerica, "CA": RegionNorthAmerica, "MX": RegionNorthAmerica,
	"BZ": RegionNorthAmerica, "GT": RegionNorthAmerica, "SV": RegionNorthAmerica,
	"HN": RegionNorthAmerica, "NI": RegionNorthAmerica, "CR": RegionNorthAmerica,
	"PA": RegionNorthAmerica, "CU": RegionNorthAmerica, "JM": RegionNorthAmerica,
	"HT": RegionNorthAmerica, "DO": RegionNorthAmerica, "TT": RegionNorthAmerica,
	"BS": RegionNorthAmerica, "BB": RegionNorthAmerica, "AG": RegionNorthAmerica,
	"DM": RegionNorthAmerica, "KN": RegionNorthAmerica, "LC": RegionNorthAmerica,
	"VC": RegionNorthAmerica, "GD": RegionNorthAmerica,
	// US territories
	"PR": RegionNorthAmerica, "VI": RegionNorthAmerica, "GU": RegionNorthAmerica,
	"MP": RegionNorthAmerica, "AS": RegionNorthAmerica, "UM": RegionNorthAmerica,
	// Other North America
	"BM": RegionNorthAmerica, "KY": RegionNorthAmerica, "TC": RegionNorthAmerica,
	"VG": RegionNorthAmerica, "AI": RegionNorthAmerica, "MS": RegionNorthAmerica,
	"GL": RegionNorthAmerica, "AW": RegionNorthAmerica, "BQ": RegionNorthAmerica,
	"CW": RegionNorthAmerica, "GP": RegionNorthAmerica, "MQ": RegionNorthAmerica,
	"BL": RegionNorthAmerica, "MF": RegionNorthAmerica, "PM": RegionNorthAmerica,
	"SX": RegionNorthAmerica,

	// South America
	"BR": RegionSouthAmerica, "AR": RegionSouthAmerica, "CL": RegionSouthAmerica,
	"CO": RegionSouthAmerica, "PE": RegionSouthAmerica, "VE": RegionSouthAmerica,
	"EC": RegionSouthAmerica, "BO": RegionSouthAmerica, "PY": RegionSouthAmerica,
	"UY": RegionSouthAmerica, "GY": RegionSouthAmerica, "SR": RegionSouthAmerica,
	"GF": RegionSouthAmerica, "FK": RegionSouthAmerica,
	"GS": RegionSouthAmerica, "BV": RegionSouthAmerica,

	// Europe
	"GB": RegionEurope, "IE": RegionEurope, "FR": RegionEurope, "DE": RegionEurope,
	"IT": RegionEurope, "ES": RegionEurope, "PT": RegionEurope, "NL": RegionEurope,
	"BE": RegionEurope, "AT": RegionEurope, "CH": RegionEurope, "SE": RegionEurope,
	"NO": RegionEurope, "DK": RegionEurope, "FI": RegionEurope, "PL": RegionEurope,
	"CZ": RegionEurope, "HU": RegionEurope, "RO": RegionEurope, "BG": RegionEurope,
	"GR": RegionEurope, "HR": RegionEurope, "SI": RegionEurope, "SK": RegionEurope,
	"LT": RegionEurope, "LV": RegionEurope, "EE": RegionEurope,
	"LU": RegionEurope, "MT": RegionEurope, "CY": RegionEurope, "IS": RegionEurope,
	"LI": RegionEurope, "MC": RegionEurope, "SM": RegionEurope, "VA": RegionEurope,
	"AD": RegionEurope, "FO": RegionEurope, "GG": RegionEurope, "JE": RegionEurope,
	"IM": RegionEurope, "AX": RegionEurope, "SJ": RegionEurope, "GI": RegionEurope,
	// Balkans
	"BA": RegionEurope, "RS": RegionEurope, "ME": RegionEurope, "MK": RegionEurope,
	"AL": RegionEurope, "XK": RegionEurope, // Kosovo not in ISO but kept for completeness
	// Eastern Europe
	"BY": RegionEurope, "UA": RegionEurope, "MD": RegionEurope, "RU": RegionEurope,
	// Turkey (assigned to Europe for map purposes)
	"TR": RegionEurope,

	// Africa
	"ZA": RegionAfrica, "NG": RegionAfrica, "EG": RegionAfrica, "KE": RegionAfrica,
	"ET": RegionAfrica, "GH": RegionAfrica, "TZ": RegionAfrica, "DZ": RegionAfrica,
	"MA": RegionAfrica, "UG": RegionAfrica, "SD": RegionAfrica,
	"CM": RegionAfrica, "CI": RegionAfrica, "NE": RegionAfrica, "BF": RegionAfrica,
	"ML": RegionAfrica, "MW": RegionAfrica, "ZW": RegionAfrica, "ZM": RegionAfrica,
	"SN": RegionAfrica, "TD": RegionAfrica, "SO": RegionAfrica, "RW": RegionAfrica,
	"BJ": RegionAfrica, "BI": RegionAfrica, "TG": RegionAfrica, "SL": RegionAfrica,
	"CG": RegionAfrica, "LY": RegionAfrica, "LR": RegionAfrica, "MR": RegionAfrica,
	"ER": RegionAfrica, "KM": RegionAfrica, "MU": RegionAfrica, "DJ": RegionAfrica,
	"SC": RegionAfrica, "CV": RegionAfrica, "ST": RegionAfrica, "SS": RegionAfrica,
	"CF": RegionAfrica, "GQ": RegionAfrica, "GW": RegionAfrica, "GA": RegionAfrica,
	"BW": RegionAfrica, "LS": RegionAfrica, "NA": RegionAfrica, "SZ": RegionAfrica,
	"MG": RegionAfrica, "YT": RegionAfrica, "RE": RegionAfrica, "SH": RegionAfrica,
	"TF": RegionAfrica, "IO": RegionAfrica,
	"AO": RegionAfrica, "CD": RegionAfrica, "GM": RegionAfrica, "GN": RegionAfrica,
	"MZ": RegionAfrica, "TN": RegionAfrica, "EH": RegionAfrica,

	// Asia
	"CN": RegionAsia, "IN": RegionAsia, "JP": RegionAsia, "ID": RegionAsia,
	"PK": RegionAsia, "BD": RegionAsia, "VN": RegionAsia, "PH": RegionAsia,
	"TH": RegionAsia, "MY": RegionAsia, "KR": RegionAsia, "IR": RegionAsia,
	"IQ": RegionAsia, "SA": RegionAsia, "AF": RegionAsia,
	"UZ": RegionAsia, "MM": RegionAsia, "KH": RegionAsia, "LK": RegionAsia,
	"NP": RegionAsia, "LA": RegionAsia, "MN": RegionAsia, "KZ": RegionAsia,
	"KG": RegionAsia, "TJ": RegionAsia, "TM": RegionAsia, "BT": RegionAsia,
	"BN": RegionAsia, "TL": RegionAsia, "MV": RegionAsia, "SG": RegionAsia,
	"HK": RegionAsia, "MO": RegionAsia, "TW": RegionAsia, "KP": RegionAsia,
	"IL": RegionAsia, "PS": RegionAsia, "JO": RegionAsia, "LB": RegionAsia,
	"SY": RegionAsia, "YE": RegionAsia, "OM": RegionAsia, "QA": RegionAsia,
	"KW": RegionAsia, "BH": RegionAsia, "AE": RegionAsia,
	// Caucasus (assigned to Asia)
	"GE": RegionAsia, "AM": RegionAsia, "AZ": RegionAsia,

	// Oceania
	"AU": RegionOceania, "NZ": RegionOceania, "PG": RegionOceania, "FJ": RegionOceania,
	"SB": RegionOceania, "VU": RegionOceania, "NC": RegionOceania, "PF": RegionOceania,
	"WS": RegionOceania, "TO": RegionOceania, "FM": RegionOceania, "KI": RegionOceania,
	"MH": RegionOceania, "PW": RegionOceania, "TV": RegionOceania, "NR": RegionOceania,
	"CK": RegionOceania, "NU": RegionOceania, "TK": RegionOceania, "WF": RegionOceania,
	"PN": RegionOceania, "NF": RegionOceania, "CX": RegionOceania, "CC": RegionOceania,
	"HM": RegionOceania,
}

// RegionOf returns the region for a country code. Every valid country has a
// region; the zero Region means unknown (should not happen for known countries).
func RegionOf(code string) Region {
	if r, ok := CountryRegion[code]; ok {
		return r
	}
	return ""
}

// RegionViewBox returns the SVG viewBox [x, y, width, height] for a region
// preset, in the map's 880x440 coordinate space. Every preset keeps the 2:1
// map aspect so the panel does not letterbox when it zooms.
func RegionViewBox(r Region) [4]float64 {
	switch r {
	case RegionWorld:
		return [4]float64{0, 0, 880, 440}
	case RegionNorthAmerica:
		return [4]float64{20, 35, 320, 160}
	case RegionSouthAmerica:
		return [4]float64{147, 198, 300, 150}
	case RegionEurope:
		return [4]float64{400, 40, 180, 90}
	case RegionAfrica:
		return [4]float64{331, 142, 300, 150}
	case RegionAsia:
		return [4]float64{500, 39, 380, 190}
	case RegionOceania:
		return [4]float64{680, 240, 200, 100}
	}
	return [4]float64{0, 0, 880, 440}
}
