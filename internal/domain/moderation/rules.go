// Package moderation holds the pure rule checks that decide a submission's
// first status (FR-MOD-1, FR-MOD-2): link count, blocked terms and repeated
// text. Model classification, flags and admin actions live in the moderation
// service and arrive in M2. No I/O.
package moderation

import (
	"regexp"
	"strings"
)

// Decision is the outcome of the rule pass.
type Decision int

const (
	// Publish means no rule objected; the pain becomes publicly visible.
	Publish Decision = iota
	// Hold means a human should review before publishing.
	Hold
	// Reject means the submission must not be published.
	Reject
)

// String names the decision for logs and tests.
func (d Decision) String() string {
	switch d {
	case Publish:
		return "publish"
	case Hold:
		return "hold"
	case Reject:
		return "reject"
	default:
		return "unknown"
	}
}

// Limits bounds the rules. Defaults come from the service; zero values mean
// "use the built-in default".
type Limits struct {
	// MaxLinks is the number of links allowed before a pain is held.
	MaxLinks int
	// MinUniqueRatio is the lowest distinct-word ratio accepted before a pain is
	// held as repeated filler text.
	MinUniqueRatio float64
}

// Defaults mirror the PRD: at most two links, and repeated filler is held.
func (l Limits) withDefaults() Limits {
	if l.MaxLinks == 0 {
		l.MaxLinks = 2
	}
	if l.MinUniqueRatio == 0 {
		l.MinUniqueRatio = 0.40
	}
	return l
}

// Result reports the decision and the reasons that produced it.
type Result struct {
	Decision Decision
	Reasons  []string
}

// linkRe finds http(s) links and bare www. hosts.
var linkRe = regexp.MustCompile(`(?i)\bhttps?://\S+|\bwww\.\S+`)

// wordRe splits on anything that is not a letter or digit.
var wordRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// Evaluate applies the rules in severity order. A self-harm match is held
// immediately and takes precedence over every other rule, so the submission
// goes to the safe-handling path and is never published (PRD FR-MOD-8). A
// reject reason then wins over a hold reason; otherwise the pain is published.
func Evaluate(body string, blockedTerms []string, limits Limits) Result {
	limits = limits.withDefaults()

	if ContainsSelfHarm(body) {
		return Result{Decision: Hold, Reasons: []string{ReasonSelfHarm}}
	}

	var holdReasons, rejectReasons []string

	if links := len(linkRe.FindAllString(body, -1)); links > limits.MaxLinks {
		holdReasons = append(holdReasons, "too many links")
	}

	if term, ok := containsBlocked(body, blockedTerms); ok {
		rejectReasons = append(rejectReasons, "blocked term: "+term)
	}

	if repeatedText(body, limits.MinUniqueRatio) {
		holdReasons = append(holdReasons, "repeated text")
	}

	switch {
	case len(rejectReasons) > 0:
		return Result{Decision: Reject, Reasons: rejectReasons}
	case len(holdReasons) > 0:
		return Result{Decision: Hold, Reasons: holdReasons}
	default:
		return Result{Decision: Publish}
	}
}

// containsBlocked reports the first blocked term present in body. Matching is
// case-insensitive and bounded so a term never fires inside a longer word.
func containsBlocked(body string, terms []string) (string, bool) {
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		bounded := `(?i)(^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(term) + `([^\p{L}\p{N}_]|$)`
		re, err := regexp.Compile(bounded)
		if err != nil {
			continue
		}
		if re.MatchString(body) {
			return strings.ToLower(term), true
		}
	}
	return "", false
}

// repeatedText reports whether the body is dominated by a small vocabulary or
// repeats the same sentence, a common spam shape.
func repeatedText(body string, minUniqueRatio float64) bool {
	words := wordRe.Split(strings.ToLower(body), -1)
	nonEmpty := words[:0]
	for _, w := range words {
		if w != "" {
			nonEmpty = append(nonEmpty, w)
		}
	}
	if len(nonEmpty) >= 20 {
		unique := make(map[string]struct{}, len(nonEmpty))
		for _, w := range nonEmpty {
			unique[w] = struct{}{}
		}
		if float64(len(unique))/float64(len(nonEmpty)) < minUniqueRatio {
			return true
		}
	}

	sentences := splitSentences(body)
	seen := make(map[string]struct{}, len(sentences))
	for _, s := range sentences {
		if len(s) < 10 {
			continue
		}
		if _, dup := seen[s]; dup {
			return true
		}
		seen[s] = struct{}{}
	}
	return false
}

func splitSentences(body string) []string {
	raw := strings.FieldsFunc(body, func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == '\n'
	})
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
