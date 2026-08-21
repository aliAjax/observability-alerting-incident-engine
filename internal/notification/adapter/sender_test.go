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

func TestSenderPropagatesRequestCancellation(t *testing.T) {
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
