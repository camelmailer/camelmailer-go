package camelmailer

// Integration roundtrip against a real CamelMailer instance. Skipped
// unless CAMELMAILER_API_KEY is set (and therefore never active in CI).
//
//	CAMELMAILER_API_KEY=cm_… \
//	CAMELMAILER_BASE_URL=https://mail.example.com \
//	CAMELMAILER_TEST_FROM=verified@your-domain.com \
//	CAMELMAILER_TEST_TO=inbox@example.com \
//	go test -run TestIntegration -v ./...

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func integrationClient(t *testing.T) *Client {
	t.Helper()
	apiKey := os.Getenv("CAMELMAILER_API_KEY")
	if apiKey == "" {
		t.Skip("CAMELMAILER_API_KEY not set; skipping integration test")
	}
	opts := []Option{}
	if baseURL := os.Getenv("CAMELMAILER_BASE_URL"); baseURL != "" {
		opts = append(opts, WithBaseURL(baseURL))
	}
	return NewClient(apiKey, opts...)
}

func TestIntegrationRoundtrip(t *testing.T) {
	client := integrationClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pong, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Logf("authenticated against server %q (id %d)", pong.Server, pong.ServerID)

	if _, err := client.Streams.List(ctx); err != nil {
		t.Fatalf("streams.list: %v", err)
	}
	if _, err := client.Templates.List(ctx); err != nil {
		t.Fatalf("templates.list: %v", err)
	}
	if _, err := client.Stats.Get(ctx, nil); err != nil {
		t.Fatalf("stats.get: %v", err)
	}
	if _, err := client.Stats.Deliveries(ctx); err != nil {
		t.Fatalf("stats.deliveries: %v", err)
	}
	if _, err := client.Bounces.List(ctx, nil); err != nil {
		t.Fatalf("bounces.list: %v", err)
	}
	if _, err := client.DMARC.Summary(ctx, nil); err != nil {
		t.Fatalf("dmarc.summary: %v", err)
	}

	from := os.Getenv("CAMELMAILER_TEST_FROM")
	to := os.Getenv("CAMELMAILER_TEST_TO")
	if from == "" || to == "" {
		t.Log("CAMELMAILER_TEST_FROM / CAMELMAILER_TEST_TO not set; skipping send roundtrip")
		return
	}

	sent, err := client.Emails.Send(ctx, &SendEmailRequest{
		From:     Address{Email: from},
		To:       []Address{{Email: to}},
		Subject:  fmt.Sprintf("camelmailer-go integration test %d", time.Now().Unix()),
		TextBody: "Sent by the camelmailer-go integration test.",
		Tag:      "camelmailer-go-integration",
	})
	if err != nil {
		t.Fatalf("emails.send: %v", err)
	}
	if sent.MessageID == 0 {
		t.Fatalf("send returned no message id: %+v", sent)
	}

	email, err := client.Emails.Get(ctx, sent.MessageID)
	if err != nil {
		t.Fatalf("emails.get: %v", err)
	}
	if email.Message.ID != sent.MessageID {
		t.Errorf("get returned message %d, want %d", email.Message.ID, sent.MessageID)
	}
	t.Logf("sent and fetched message %d (status %q)", email.Message.ID, email.Message.Status)
}
