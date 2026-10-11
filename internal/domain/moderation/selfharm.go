package moderation

import (
	"regexp"
	"strings"
)

// ReasonSelfHarm is the reason recorded when a submission is held for safety.
const ReasonSelfHarm = "self-harm"

// selfHarmPhrases is a deliberately narrow, conservative lexicon. It is a
// Phase 1 safety floor (PRD FR-MOD-8): anything it matches is held and never
// published, so a false negative is the dangerous direction and a false
// positive merely sends ordinary text to human review. The M2 model classifier
// augments this list; it does not replace the hold.
var selfHarmPhrases = []string{
	"kill myself",
	"killing myself",
	"end my life",
	"ending my life",
	"take my own life",
	"take my life",
	"want to die",
	"wish i was dead",
	"wish i were dead",
	"better off dead",
	"no reason to live",
	"nothing to live for",
	"don't want to live",
	"do not want to live",
	"hurt myself",
	"hurting myself",
	"harm myself",
	"harming myself",
	"self harm",
	"cut myself",
	"cutting myself",
	"overdose",
	"suicide",
	"suicidal",
}

// phraseRe matches a phrase bounded by non-word characters.
func phraseRe(phrase string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(phrase) + `([^\p{L}\p{N}_]|$)`)
}

// ContainsSelfHarm reports whether text carries a self-harm phrase. Matching is
// case-insensitive; hyphens are treated as spaces so "self-harm" and "self harm"
// both match.
func ContainsSelfHarm(text string) bool {
	normalized := strings.ReplaceAll(strings.ToLower(text), "-", " ")
	for _, phrase := range selfHarmPhrases {
		if phraseRe(strings.ReplaceAll(phrase, "-", " ")).MatchString(normalized) {
			return true
		}
	}
	return false
}
