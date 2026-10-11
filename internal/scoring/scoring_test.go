package scoring

import (
	"testing"
	"time"
)

// TestUrgencyPRDExamples reproduces the worked examples in PRD section 9.2.
func TestUrgencyPRDExamples(t *testing.T) {
	cases := []struct {
		name string
		in   Signals
		want int
	}{
		{
			name: "daily annoying pay$10 10 votes",
			in:   Signals{Frequency: FrequencyDaily, Disruption: DisruptionAnnoying, WouldPay: true, MonthlyAmount: 10, Votes: 10},
			want: 76,
		},
		{
			name: "daily ruins week pay$20+ 50 votes",
			in:   Signals{Frequency: FrequencyDaily, Disruption: DisruptionRuinsWeek, WouldPay: true, MonthlyAmount: 20, Votes: 50},
			want: 100,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Urgency(tc.in); got != tc.want {
				t.Errorf("Urgency(%+v) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestUrgencyNeutralOneSentence(t *testing.T) {
	// Unset submitter fields use neutral values (PRD 9.1); no votes.
	got := Urgency(Signals{})
	if got != 18 {
		t.Errorf("Urgency(empty) = %d, want 18", got)
	}
	if got <= 0 {
		t.Error("a one-sentence pain should still score above zero")
	}
}

func TestUrgencyBounds(t *testing.T) {
	max := Urgency(Signals{
		Frequency: FrequencyDaily, Disruption: DisruptionRuinsWeek,
		WouldPay: true, MonthlyAmount: 1000, Votes: 1_000_000,
	})
	if max != 100 {
		t.Errorf("max urgency = %d, want 100", max)
	}
	min := Urgency(Signals{Frequency: FrequencySometimes, Disruption: DisruptionMild})
	if min < 0 || min > 100 {
		t.Errorf("urgency out of range: %d", min)
	}
}

func TestPayNormalisation(t *testing.T) {
	if n := normalizePay(true, 0); n != payYesNoAmount {
		t.Errorf("would pay without amount = %v, want %v", n, payYesNoAmount)
	}
	if n := normalizePay(true, 10); n != 0.8 {
		t.Errorf("would pay $10 = %v, want 0.8", n)
	}
	if n := normalizePay(true, 20); n != 1.0 {
		t.Errorf("would pay $20 = %v, want 1.0", n)
	}
	if n := normalizePay(true, 500); n != 1.0 {
		t.Errorf("would pay $500 should cap at 1.0, got %v", n)
	}
	if n := normalizePay(false, 500); n != 0 {
		t.Errorf("would not pay should be 0, got %v", n)
	}
}

func TestCrowdNormalisation(t *testing.T) {
	if n := normalizeCrowd(0); n != 0 {
		t.Errorf("0 votes = %v, want 0", n)
	}
	if n := normalizeCrowd(50); n != 1.0 {
		t.Errorf("50 votes = %v, want 1.0", n)
	}
	if n := normalizeCrowd(5000); n != 1.0 {
		t.Errorf("crowd should cap at 1.0, got %v", n)
	}
	if a, b := normalizeCrowd(5), normalizeCrowd(25); !(b > a) {
		t.Errorf("crowd should increase with votes: %v vs %v", a, b)
	}
}

func TestHotDecaysWithAge(t *testing.T) {
	younger := Hot(10, 2, 80, time.Hour)
	older := Hot(10, 2, 80, 24*time.Hour)
	if younger <= older {
		t.Errorf("younger pain should rank hotter: %v vs %v", younger, older)
	}
	if Hot(0, 0, 0, time.Hour) != 0 {
		t.Error("empty signals should rank 0 by raw numerator")
	}
}
