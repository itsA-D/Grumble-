package pain

import (
	"strings"
	"unicode"
)

// MaxSlugWords bounds how many leading words are taken from the body.
const MaxSlugWords = 8

// MaxSlugLen bounds the words part of a slug before the unique suffix.
const MaxSlugLen = 60

// Slugify turns free text into a URL-safe, human-readable slug body: lower-case
// ASCII letters and digits separated by single hyphens. Unicode letters are
// folded to their ASCII form where possible and dropped otherwise (NFR-I18N-1).
// The caller appends a unique suffix (FR-SUB-8).
func Slugify(text string) string {
	var b strings.Builder
	lastHyphen := true // suppress a leading hyphen
	words := 0

	for _, r := range text {
		switch r {
		case ' ', '\t', '\n', '\r', '-', '_', '/':
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		default:
			folded := foldASCII(r)
			if folded == 0 {
				continue
			}
			if lastHyphen {
				if words >= MaxSlugWords {
					return strings.Trim(b.String(), "-")
				}
				words++
			}
			b.WriteRune(folded)
			lastHyphen = false
			if b.Len() >= MaxSlugLen {
				return strings.Trim(b.String(), "-")
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// foldASCII maps a rune to a lower-case ASCII letter or digit, folding common
// Latin accented characters. It returns 0 when the rune has no ASCII form.
func foldASCII(r rune) rune {
	switch r {
	case 'à', 'á', 'â', 'ã', 'ä', 'å', 'À', 'Á', 'Â', 'Ã', 'Ä', 'Å':
		return 'a'
	case 'è', 'é', 'ê', 'ë', 'È', 'É', 'Ê', 'Ë':
		return 'e'
	case 'ì', 'í', 'î', 'ï', 'Ì', 'Í', 'Î', 'Ï':
		return 'i'
	case 'ò', 'ó', 'ô', 'õ', 'ö', 'Ò', 'Ó', 'Ô', 'Õ', 'Ö':
		return 'o'
	case 'ù', 'ú', 'û', 'ü', 'Ù', 'Ú', 'Û', 'Ü':
		return 'u'
	case 'ç', 'Ç':
		return 'c'
	case 'ñ', 'Ñ':
		return 'n'
	case 'ý', 'ÿ', 'Ý':
		return 'y'
	case 'ß':
		return 's'
	}
	switch {
	case r >= 'a' && r <= 'z':
		return r
	case r >= '0' && r <= '9':
		return r
	case r >= 'A' && r <= 'Z':
		return unicode.ToLower(r)
	default:
		return 0
	}
}
