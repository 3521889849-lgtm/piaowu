package metrics

import "testing"

func TestCollector_Snapshot(t *testing.T) {
	c := NewCollector(5)
	c.RecordDecision(DecisionPass, 1000000)
	c.RecordDecision(DecisionReject, 2000000)

	c.RecordManualOutcome(DecisionPass, DecisionPass)
	c.RecordManualOutcome(DecisionReject, DecisionPass)

	s := c.Snapshot()
	if s.TotalCount != 2 {
		t.Fatalf("want total 2, got %d", s.TotalCount)
	}
	if s.AvgLatencyMs <= 0 {
		t.Fatalf("want avg latency > 0")
	}
	if s.ManualTotal != 2 {
		t.Fatalf("want manual total 2, got %d", s.ManualTotal)
	}
	if s.Accuracy <= 0 {
		t.Fatalf("want accuracy > 0")
	}
}
