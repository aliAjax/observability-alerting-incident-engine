package domain

import "testing"

func TestR004ResolvedBackToFiringEdge(t *testing.T) {
	if err := ValidateTransition(StatusResolved, StatusFiring); err != nil {
		t.Fatalf("resolved to firing should be allowed, got %v", err)
	}
}
