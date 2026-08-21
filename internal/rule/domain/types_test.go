package domain

import "testing"

func TestR002PausedRuleExcluded(t *testing.T) {
	rule := Rule{Enabled: true, Mode: ModePaused}
	if rule.Active() {
		t.Fatal("paused rule should not be active")
	}
}
