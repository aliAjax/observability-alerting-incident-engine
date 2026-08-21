package config

import (
	"testing"
	"time"
)

func TestDefaultValid(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
	if cfg.Evaluator.Interval <= 0 {
		t.Fatal("expected positive evaluator interval")
	}
}

func TestLoadFromBytes(t *testing.T) {
	cfg := Default()
	cfg.Evaluator.Interval = time.Second
	if cfg.Evaluator.Interval != time.Second {
		t.Fatal("unexpected duration")
	}
}
