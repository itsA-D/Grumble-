package moderation

import "testing"

func TestEvaluateCleanPublishes(t *testing.T) {
	res := Evaluate("My landlord ignores every repair request for months at a time.", nil, Limits{})
	if res.Decision != Publish {
		t.Fatalf("decision = %s (%v), want publish", res.Decision, res.Reasons)
	}
}

func TestEvaluateTooManyLinksHeld(t *testing.T) {
	body := "See https://a.example and https://b.example and https://c.example for details."
	res := Evaluate(body, nil, Limits{})
	if res.Decision != Hold {
		t.Fatalf("decision = %s, want hold", res.Decision)
	}
}

func TestEvaluateTwoLinksPublishes(t *testing.T) {
	body := "See https://a.example and https://b.example for the two sources."
	if res := Evaluate(body, nil, Limits{}); res.Decision != Publish {
		t.Fatalf("decision = %s, want publish", res.Decision)
	}
}

func TestEvaluateBlockedTermRejected(t *testing.T) {
	res := Evaluate("This page is a scam scheme to steal all the money.", []string{"scam"}, Limits{})
	if res.Decision != Reject {
		t.Fatalf("decision = %s, want reject", res.Decision)
	}
}

func TestEvaluateBlockedTermIsWordBounded(t *testing.T) {
	// "scam" inside "scampi" must not fire.
	res := Evaluate("I ordered scampi at the restaurant and it arrived cold.", []string{"scam"}, Limits{})
	if res.Decision != Publish {
		t.Fatalf("decision = %s, want publish (no false positive)", res.Decision)
	}
}

func TestEvaluateRepeatedSentenceHeld(t *testing.T) {
	body := "Buy now cheap deals cheap deals. Buy now cheap deals cheap deals."
	res := Evaluate(body, nil, Limits{})
	if res.Decision != Hold {
		t.Fatalf("decision = %s, want hold", res.Decision)
	}
}

func TestEvaluateRejectBeatsHold(t *testing.T) {
	body := "spam spam spam https://a.example https://b.example https://c.example spam spam spam"
	res := Evaluate(body, []string{"spam"}, Limits{})
	if res.Decision != Reject {
		t.Fatalf("decision = %s, want reject to win over hold", res.Decision)
	}
}

func TestDecisionString(t *testing.T) {
	if Publish.String() != "publish" || Hold.String() != "hold" || Reject.String() != "reject" {
		t.Fatal("decision names drifted")
	}
}

func TestSelfHarmIsHeld(t *testing.T) {
	cases := []string{
		"I just want to die and nothing helps anymore.",
		"Sometimes I think about ending my life.",
		"I have been cutting myself again this week.",
		"These thoughts of suicide will not stop.",
	}
	for _, body := range cases {
		res := Evaluate(body, nil, Limits{})
		if res.Decision != Hold {
			t.Fatalf("%q: decision = %s, want hold", body, res.Decision)
		}
		if len(res.Reasons) != 1 || res.Reasons[0] != ReasonSelfHarm {
			t.Fatalf("%q: reasons = %v, want [%s]", body, res.Reasons, ReasonSelfHarm)
		}
	}
}

func TestSelfHarmBeatsBlockedAndLinks(t *testing.T) {
	body := "free crypto https://a.example https://b.example https://c.example I want to die"
	res := Evaluate(body, []string{"free crypto"}, Limits{})
	if res.Decision != Hold {
		t.Fatalf("decision = %s, want hold (self-harm takes precedence)", res.Decision)
	}
	if res.Reasons[0] != ReasonSelfHarm {
		t.Fatalf("reasons = %v, want self-harm first", res.Reasons)
	}
}

func TestSelfHarmDoesNotMatchInsideWords(t *testing.T) {
	if ContainsSelfHarm("The word suicideideation is not a phrase I would use casually.") {
		t.Fatal("substring match leaked through word boundaries")
	}
	if ContainsSelfHarm("We had to kill time before the train arrived.") {
		t.Fatal("benign phrase flagged as self-harm")
	}
}
