// Package scoring holds the pure urgency and hot-rank functions. Weights and
// normalisation constants live here only (ARCHITECTURE.md invariant 8); nothing
// else hard-codes a weight. It imports nothing from the module and does no I/O.
package scoring

// Version identifies the current weight set; stored in scoring_versions so a
// score can be traced to the weights that produced it.
const Version = 1

// Urgency weights (PRD section 9.2). They must sum to 1.
const (
	WeightFrequency  = 0.30
	WeightDisruption = 0.25
	WeightPay        = 0.25
	WeightCrowd      = 0.20
)

// Normalised submitter and crowd values (PRD section 9.1).
const (
	freqDaily     = 1.00
	freqWeekly    = 0.60
	freqSometimes = 0.30
	freqUnset     = 0.30

	disruptionRuinsWeek = 1.00
	disruptionAnnoying  = 0.55
	disruptionMild      = 0.20
	disruptionUnset     = 0.35

	payNo            = 0.0
	payYesNoAmount   = 0.60
	payAmountBase    = 0.60
	payAmountScale   = 0.40
	payAmountRefUSD  = 20.0
	crowdVoteRef     = 50.0
	urgencyPrecision = 100.0
)

// Frequency and disruption string values accepted by Signals. They match the
// domain enums; a test asserts the two stay aligned.
const (
	FrequencyDaily     = "daily"
	FrequencyWeekly    = "weekly"
	FrequencySometimes = "sometimes"

	DisruptionMild      = "mild"
	DisruptionAnnoying  = "annoying"
	DisruptionRuinsWeek = "ruins_my_week"
)

// Weights is the current weight set, for persistence in scoring_versions.
type Weights struct {
	Version    int
	Frequency  float64
	Disruption float64
	Pay        float64
	Crowd      float64
}

// Current returns the active weights.
func Current() Weights {
	return Weights{
		Version:    Version,
		Frequency:  WeightFrequency,
		Disruption: WeightDisruption,
		Pay:        WeightPay,
		Crowd:      WeightCrowd,
	}
}
