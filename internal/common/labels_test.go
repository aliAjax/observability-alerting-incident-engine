package common

import "testing"

func TestR005MergeEmptyLabels(t *testing.T) {
	var labels Labels
	merged := labels.Merge(Labels{"region": "cn-east", "team": "platform"})
	if merged["region"] != "cn-east" || merged["team"] != "platform" {
		t.Fatalf("merged labels are incomplete: %#v", merged)
	}
}

func TestR005CloneLabelsDeepCopy(t *testing.T) {
	original := Labels{"team": "platform"}
	clone := original.Clone()
	clone["team"] = "observability"
	if original["team"] != "platform" {
		t.Fatal("Clone should not alias the receiver")
	}
}

func TestR005EmptyLabelsString(t *testing.T) {
	var labels Labels
	if got := labels.String(); got != "{}" {
		t.Fatalf("expected empty map string, got %q", got)
	}
}

func TestR005EmptyMatcherOnEmpty(t *testing.T) {
	var labels Labels
	if !labels.Match(Labels{}) {
		t.Fatal("an empty matcher should match a nil label set")
	}
}
