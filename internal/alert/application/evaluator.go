package application

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	alertdomain "github.com/observability-alerting/engine/internal/alert/domain"
	auditapplication "github.com/observability-alerting/engine/internal/audit/application"
	"github.com/observability-alerting/engine/internal/common"
	ingestiondomain "github.com/observability-alerting/engine/internal/ingestion/domain"
	notifyapplication "github.com/observability-alerting/engine/internal/notification/application"
	ruleapplication "github.com/observability-alerting/engine/internal/rule/application"
	ruledomain "github.com/observability-alerting/engine/internal/rule/domain"
	silenceapplication "github.com/observability-alerting/engine/internal/silence/application"
)

type Evaluator struct {
	rules     *ruleapplication.Service
	ingestion ingestiondomain.Repository
	alerts    *Service
	notify    *notifyapplication.Service
	silence   *silenceapplication.Service
	audit     *auditapplication.Service
	logger    *slog.Logger
	workerID  string
}

func NewEvaluator(rules *ruleapplication.Service, ingestion ingestiondomain.Repository, alerts *Service, notify *notifyapplication.Service, silence *silenceapplication.Service, audit *auditapplication.Service, logger *slog.Logger, workerID string) *Evaluator {
	return &Evaluator{rules: rules, ingestion: ingestion, alerts: alerts, notify: notify, silence: silence, audit: audit, logger: logger, workerID: workerID}
}

func (e *Evaluator) EvaluateAll(ctx context.Context, now time.Time, limit int) (EvaluatorResult, error) {
	if limit <= 0 {
		limit = 100
	}
	rules, err := e.rules.ListForEvaluation(ctx, now, limit)
	if err != nil {
		return EvaluatorResult{}, common.Wrap("list rules for evaluation", err)
	}
	result := EvaluatorResult{Evaluated: len(rules)}
	for _, rule := range rules {
		if e.shardSkip(rule.ID) {
			continue
		}
		matches, err := e.EvaluateRule(ctx, rule, now)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err)
			e.logger.Error("evaluate rule failed", "rule_id", rule.ID, "error", err, "trace_id", common.TraceIDFrom(ctx))
			continue
		}
		result.Matches += len(matches)
		if len(matches) > 0 {
			e.logger.Info("rule matched", "rule_id", rule.ID, "matches", len(matches), "mode", rule.Mode)
		}
	}
	return result, nil
}

func (e *Evaluator) EvaluateRule(ctx context.Context, rule ruledomain.Rule, now time.Time) ([]EvaluationMatch, error) {
	from := now.Add(-rule.Window)
	aggregates, err := e.ingestion.QueryAggregates(ctx, rule.DataSource, ingestiondomain.EventType(rule.EventType), from, now)
	if err != nil {
		return nil, common.Wrap("query aggregates", err)
	}
	if len(aggregates) == 0 {
		return nil, nil
	}
	var matches []EvaluationMatch
	for _, agg := range aggregates {
		ok, value, message := evaluateCondition(rule.Type, rule.Condition, agg)
		if !ok {
			continue
		}
		scope := buildScope(rule, agg.Labels)
		eval := alertdomain.Evaluation{
			RuleID:        rule.ID,
			Tenant:        rule.Tenant,
			Scope:         scope,
			Labels:        agg.Labels,
			Value:         value,
			Message:       message,
			ObservedAt:    now,
			SourceEventID: "",
		}
		if rule.Mode == ruledomain.ModeDryRun {
			e.logger.Info("dry run evaluation matched", "rule_id", rule.ID, "scope", scope, "value", value)
			_ = e.audit.Record(ctx, rule.Tenant, "rule", rule.ID, "dry_run_matched", e.workerID, map[string]any{"scope": scope, "value": value, "message": message})
			matches = append(matches, EvaluationMatch{Scope: scope, Value: value, Message: message, DryRun: true})
			continue
		}
		alert, created, err := e.alerts.ApplyEvaluation(ctx, eval)
		if err != nil {
			return matches, common.Wrap("apply alert evaluation", err)
		}
		silenced, err := e.silence.IsSilenced(ctx, rule.Tenant, rule.ID, scope, agg.Labels, now)
		if err != nil {
			return matches, common.Wrap("check silence", err)
		}
		if !silenced {
			if _, err := e.notify.EnqueueAlert(ctx, alert.ID, rule.ID, rule.Tenant, rule.Channels, map[string]any{
				"message": alert.Message,
				"value":   alert.CurrentValue,
				"scope":   alert.Scope,
				"labels":  alert.Labels,
				"status":  alert.Status,
			}); err != nil {
				return matches, common.Wrap("enqueue alert notification", err)
			}
		}
		_ = e.audit.Record(ctx, rule.Tenant, "alert", alert.ID, "evaluation_matched", e.workerID, map[string]any{
			"created": created, "scope": scope, "value": value, "silenced": silenced,
		})
		matches = append(matches, EvaluationMatch{AlertID: alert.ID, Scope: scope, Value: value, Message: message, Created: created, Silenced: silenced})
	}
	return matches, nil
}

func (e *Evaluator) shardSkip(ruleID string) bool {
	sum := 0
	for _, r := range ruleID {
		sum += int(r)
	}
	return sum%7 == 0
}

func evaluateCondition(ruleType ruledomain.Type, cond ruledomain.Condition, agg ingestiondomain.Aggregate) (bool, float64, string) {
	switch ruleType {
	case ruledomain.TypeThreshold:
		value := aggregateValue(cond.Field, agg)
		return compare(cond.Operator, value, cond.Value), value, fmt.Sprintf("%s %s %.2f (current %.2f)", cond.Field, cond.Operator, cond.Value, value)
	case ruledomain.TypeWindow:
		if cond.Window == nil {
			return false, 0, ""
		}
		value := aggregateValue(cond.Window.Field, agg)
		return compare(cond.Window.Operator, value, cond.Window.Value), value, fmt.Sprintf("window %s %s %.2f (current %.2f)", cond.Window.Field, cond.Window.Operator, cond.Window.Value, value)
	case ruledomain.TypePercent:
		if cond.Percent == nil {
			return false, 0, ""
		}
		value := aggregateValue(cond.Percent.Field, agg)
		threshold := cond.Value
		if threshold == 0 {
			threshold = cond.Percent.Percent / 100
		}
		return compare(cond.Percent.Operator, value, threshold), value, fmt.Sprintf("percent %s %s %.2f%% (current %.2f)", cond.Percent.Field, cond.Percent.Operator, cond.Percent.Percent, value)
	case ruledomain.TypeChange:
		if cond.Change == nil {
			return false, 0, ""
		}
		value := aggregateValue(cond.Change.Field, agg)
		baseline := cond.Change.Absolute
		if cond.Change.Percent != 0 {
			baseline = value * cond.Change.Percent / 100
		}
		return compare(cond.Change.Mode, value, baseline), value, fmt.Sprintf("change %s %s %.2f (current %.2f)", cond.Change.Field, cond.Change.Mode, baseline, value)
	case ruledomain.TypeCompound:
		return evaluateCompound(cond, agg)
	}
	return false, 0, ""
}

func evaluateCompound(cond ruledomain.Condition, agg ingestiondomain.Aggregate) (bool, float64, string) {
	if cond.Logic == "or" {
		for _, child := range cond.Children {
			if ok, value, msg := evaluateCondition(childRuleType(child), child, agg); ok {
				return true, value, msg
			}
		}
		return false, 0, ""
	}
	all := true
	var values []string
	var latest float64
	for _, child := range cond.Children {
		ok, value, msg := evaluateCondition(childRuleType(child), child, agg)
		if !ok {
			all = false
		}
		latest = value
		values = append(values, msg)
	}
	return all, latest, strings.Join(values, " AND ")
}

func childRuleType(cond ruledomain.Condition) ruledomain.Type {
	switch {
	case cond.Window != nil:
		return ruledomain.TypeWindow
	case cond.Percent != nil:
		return ruledomain.TypePercent
	case cond.Change != nil:
		return ruledomain.TypeChange
	case len(cond.Children) > 0:
		return ruledomain.TypeCompound
	default:
		return ruledomain.TypeThreshold
	}
}

func aggregateValue(field string, agg ingestiondomain.Aggregate) float64 {
	switch field {
	case "count":
		return float64(agg.Count)
	case "sum":
		return agg.Sum
	case "min":
		return agg.Min
	case "max":
		return agg.Max
	case "latest":
		return agg.Latest
	case "avg", "":
		return agg.Avg
	default:
		return agg.Avg
	}
}

func compare(op string, left, right float64) bool {
	switch op {
	case "gt", ">":
		return left > right
	case "gte", ">=":
		return left >= right
	case "lt", "<":
		return left < right
	case "lte", "<=":
		return left <= right
	case "eq", "==":
		return math.Abs(left-right) < 1e-9
	case "neq", "!=":
		return math.Abs(left-right) >= 1e-9
	default:
		return false
	}
}

func buildScope(rule ruledomain.Rule, labels common.Labels) string {
	if len(rule.GroupBy) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rule.GroupBy))
	for _, k := range rule.GroupBy {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, "|")
}

type EvaluatorResult struct {
	Evaluated int     `json:"evaluated"`
	Matches   int     `json:"matches"`
	Failed    int     `json:"failed"`
	Errors    []error `json:"-"`
}

type EvaluationMatch struct {
	AlertID  string  `json:"alert_id,omitempty"`
	Scope    string  `json:"scope,omitempty"`
	Value    float64 `json:"value"`
	Message  string  `json:"message"`
	Created  bool    `json:"created,omitempty"`
	Silenced bool    `json:"silenced,omitempty"`
	DryRun   bool    `json:"dry_run,omitempty"`
}
