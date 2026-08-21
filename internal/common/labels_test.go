package common

import "testing"

func TestLabelsMergeOnNilReceiver(t *testing.T) {
	var labels Labels
	merged := labels.Merge(Labels{"region": "cn-east", "team": "platform"})
	if merged["region"] != "cn-east" || merged["team"] != "platform" {
		t.Fatalf("merged labels are incomplete: %#v", merged)
	}
}
