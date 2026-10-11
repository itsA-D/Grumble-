package scoring

import (
	"math"
	"time"
)

// Hot-rank constants (PRD section 9.3).
const (
	commentWeight = 0.5
	ageOffset     = 2.0
	ageExponent   = 1.5
)

// Hot returns the trending rank: higher is hotter (PRD section 9.3). It decays
// with age so recent, urgent, well-voted pains surface first.
func Hot(votes7d, comments7d, urgency int, age time.Duration) float64 {
	ageHours := age.Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	numerator := float64(votes7d) + commentWeight*float64(comments7d) + float64(urgency)/20.0
	return numerator / math.Pow(ageHours+ageOffset, ageExponent)
}
