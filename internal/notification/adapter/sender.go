package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/observability-alerting/engine/internal/notification/application"
	"github.com/observability-alerting/engine/internal/notification/domain"
)

type Sender struct {
	client *http.Client
	logger *slog.Logger
}

func NewSender(client *http.Client, logger *slog.Logger) *Sender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Sender{client: client, logger: logger}
}

func (s *Sender) Send(ctx context.Context, channel domain.Channel, template domain.Template, task domain.Task) (domain.Result, error) {
	subject, body := application.RenderTemplate(template, task)
	if channel.Type == domain.ChannelWebhook || channel.Type == domain.ChannelSlack {
		cfg := map[string]string{}
		_ = json.Unmarshal(channel.Config, &cfg)
		url := cfg["url"]
		if url == "" {
			return domain.Result{Delivered: false, Message: "webhook url is empty"}, nil
		}
		payload := map[string]any{
			"subject":    subject,
			"body":       body,
			"alert_id":   task.AlertID,
			"rule_id":    task.RuleID,
			"receivers":  task.Receivers,
			"timestamp":  time.Now().UTC(),
			"escalation": task.EscalationStep,
		}
		raw, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return domain.Result{}, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.client.Do(req)
		if err != nil {
			return domain.Result{}, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return domain.Result{Delivered: false, Message: fmt.Sprintf("webhook returned %d", resp.StatusCode)}, nil
		}
		s.logger.Info("notification sent", "channel_id", channel.ID, "alert_id", task.AlertID, "url", url)
		return domain.Result{Delivered: true}, nil
	}
	if err := ctx.Err(); err != nil {
		return domain.Result{}, err
	}
	s.logger.Info("notification rendered", "channel_type", channel.Type, "subject", subject, "body", body)
	return domain.Result{Delivered: true}, nil
}
