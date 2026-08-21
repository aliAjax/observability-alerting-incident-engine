package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/observability-alerting/engine/internal/common"
	"github.com/observability-alerting/engine/internal/rule/domain"
)

type fakeRuleRepo struct {
	getErr    error
	createErr error
}

func (f fakeRuleRepo) Create(context.Context, domain.Rule) (domain.Rule, error) {
	return domain.Rule{}, f.createErr
}
func (f fakeRuleRepo) Update(context.Context, domain.Rule) (domain.Rule, error) {
	return domain.Rule{}, nil
}
func (f fakeRuleRepo) Get(context.Context, string, string) (domain.Rule, error) {
	if f.getErr != nil {
		return domain.Rule{}, f.getErr
	}
	return domain.Rule{}, nil
}
func (f fakeRuleRepo) List(context.Context, string, int, int, map[string]string) ([]domain.Rule, int, error) {
	return nil, 0, f.getErr
}
func (f fakeRuleRepo) ListForEvaluation(context.Context, time.Time, int) ([]domain.Rule, error) {
	return nil, nil
}
func (f fakeRuleRepo) GetByName(context.Context, string, string) (domain.Rule, error) {
	return domain.Rule{}, f.getErr
}

func TestRuleGetAndListPreserveSentinel(t *testing.T) {
	repo := fakeRuleRepo{getErr: fmt.Errorf("repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Get(context.Background(), "default", "rule-1"); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Get should preserve ErrNotFound, got %v", err)
	}
	if _, err := service.List(context.Background(), "default", 10, 0, nil); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("List should preserve ErrNotFound, got %v", err)
	}
}

func TestR002RuleGetAbsentRule(t *testing.T) {
	repo := fakeRuleRepo{getErr: fmt.Errorf("repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Get(context.Background(), "default", "rule-1"); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Get should preserve ErrNotFound, got %v", err)
	}
}

func TestR002RuleCreateBadPayload(t *testing.T) {
	repo := fakeRuleRepo{createErr: fmt.Errorf("duplicate rule: %w", common.ErrAlreadyExists)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	rule := domain.Rule{Name: "cpu-high", DataSource: "api-gateway", Type: domain.TypeThreshold, Condition: domain.Condition{Field: "cpu", Operator: ">"}}

	if _, err := service.Create(context.Background(), rule); !errors.Is(err, common.ErrAlreadyExists) {
		t.Fatalf("Create should preserve ErrAlreadyExists, got %v", err)
	}
}

func TestR002RuleUpdateAbsentRule(t *testing.T) {
	repo := fakeRuleRepo{getErr: fmt.Errorf("repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.Update(context.Background(), "rule-1", domain.Rule{Tenant: "default"}); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("Update should preserve ErrNotFound, got %v", err)
	}
}

func TestR002RuleListAbsentRule(t *testing.T) {
	repo := fakeRuleRepo{getErr: fmt.Errorf("repository miss: %w", common.ErrNotFound)}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.List(context.Background(), "default", 10, 0, nil); !errors.Is(err, common.ErrNotFound) {
		t.Fatalf("List should preserve ErrNotFound, got %v", err)
	}
}

func TestR002RuleSetModeBadEnum(t *testing.T) {
	repo := fakeRuleRepo{}
	service := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	if _, err := service.SetMode(context.Background(), "default", "rule-1", domain.Mode("invalid")); !errors.Is(err, common.ErrInvalid) {
		t.Fatalf("SetMode should preserve ErrInvalid, got %v", err)
	}
}

func TestRuleNormalizeInitializesLabels(t *testing.T) {
	rule := domain.Rule{Name: "cpu-high", DataSource: "api-gateway"}
	if err := rule.Normalize(time.Now()); err != nil {
		t.Fatalf("Normalize returned error: %v", err)
	}
	if rule.Labels == nil {
		t.Fatal("Normalize should initialize Labels")
	}
}
