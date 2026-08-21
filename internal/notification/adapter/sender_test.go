package adapter

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/observability-alerting/engine/internal/notification/domain"
)

func TestR003WebhookCancelStopsSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &http.Client{}
	sender := NewSender(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	channel := domain.Channel{
		ID:      "webhook-1",
		Tenant:  "default",
		Type:    domain.ChannelWebhook,
		Enabled: true,
		Config:  json.RawMessage(`{"url":"` + server.URL + `"}`),
	}
	task := domain.Task{Payload: json.RawMessage(`{"message":"cpu high"}`)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sender.Send(ctx, channel, domain.Template{Subject: "alert", Body: "{{.message}}"}, task); err == nil {
		t.Fatal("expected canceled request to fail instead of completing")
	}
}

func TestR003EmailCancelStopsSend(t *testing.T) {
	sender := NewSender(&http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := sender.Send(ctx, domain.Channel{Type: domain.ChannelEmail}, domain.Template{Subject: "alert"}, domain.Task{}); err == nil {
		t.Fatal("expected non-webhook send to fail when the context is canceled")
	}
}
