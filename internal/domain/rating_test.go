package domain

import "testing"

func TestNextRatingDelta(t *testing.T) {
	if got := NextRatingDelta(0, true); got != 10 {
		t.Fatalf("expected 10, got %d", got)
	}
	if got := NextRatingDelta(0, false); got != -10 {
		t.Fatalf("expected -10, got %d", got)
	}
	if got := NextRatingDelta(200, true); got != 200 {
		t.Fatalf("expected 200 clamp, got %d", got)
	}
	if got := NextRatingDelta(-200, false); got != -200 {
		t.Fatalf("expected -200 clamp, got %d", got)
	}
}
