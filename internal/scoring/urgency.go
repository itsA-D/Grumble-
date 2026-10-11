package scoring

import "math"

// Signals is the raw input to the urgency score: what the submitter said plus
// how the crowd voted (PRD section 9.1).
type Signals struct {
	// Frequency is "daily", "weekly", "sometimes" or "" (unset).
	Frequency string
	// Disruption is "mild", "annoying", "ruins_my_week" or "" (unset).
	Disruption string
	// WouldPay is the submitter's willingness-to-pay flag.
	WouldPay bool
	// MonthlyAmount is USD per month; zero means "yes but unspecified".
	MonthlyAmount int
	// Votes is the current vote count.
	Votes int
}

// Urgency returns the 0 to 100 urgency score (PRD section 9.2). Unset inputs use
// neutral values so a one-sentence pain still scores.
func Urgency(s Signals) int {
	u := WeightFrequency*normalizeFrequency(s.Frequency) +
		WeightDisruption*normalizeDisruption(s.Disruption) +
		WeightPay*normalizePay(s.WouldPay, s.MonthlyAmount) +
		WeightCrowd*normalizeCrowd(s.Votes)
	return int(math.Round(urgencyPrecision * u))
}

func normalizeFrequency(v string) float64 {
	switch v {
	case FrequencyDaily:
		return freqDaily
	case FrequencyWeekly:
		return freqWeekly
	case FrequencySometimes:
		return freqSometimes
	default:
		return freqUnset
	}
}

func normalizeDisruption(v string) float64 {
	switch v {
	case DisruptionRuinsWeek:
		return disruptionRuinsWeek
	case DisruptionAnnoying:
		return disruptionAnnoying
	case DisruptionMild:
		return disruptionMild
	default:
		return disruptionUnset
	}
}

func normalizePay(wouldPay bool, monthlyAmount int) float64 {
	if !wouldPay {
		return payNo
	}
	if monthlyAmount <= 0 {
		return payYesNoAmount
	}
	return math.Min(1, payAmountBase+payAmountScale*float64(monthlyAmount)/payAmountRefUSD)
}

func normalizeCrowd(votes int) float64 {
	if votes <= 0 {
		return 0
	}
	return math.Min(1, math.Log1p(float64(votes))/math.Log1p(crowdVoteRef))
}
