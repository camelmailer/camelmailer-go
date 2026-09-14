package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func sendRequest() *SendEmailRequest {
	return &SendEmailRequest{
		From:     Address{Email: "billing@acme.com"},
		To:       []Address{{Email: "ada@example.com"}},
		Subject:  "Your receipt",
		TextBody: "Thanks.",
	}
}

func TestIdempotencyKeyTravelsAsAHeader(t *testing.T) {
	var gotKey string
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Idempotency-Key")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"message_id":1,"recipients":[]}`)
	})
	if _, err := client.Emails.Send(context.Background(), sendRequest(), WithIdempotencyKey("order-4711")); err != nil {
		t.Fatal(err)
	}
	if gotKey != "order-4711" {
		t.Errorf("Idempotency-Key = %q", gotKey)
	}
	// The body is what the server hashes for the claim, so the key must
	// not leak into it.
	for _, key := range []string{"idempotency_key", "Idempotency-Key"} {
		if _, present := gotBody[key]; present {
			t.Errorf("%s ended up in the body: %v", key, gotBody)
		}
	}
}

func TestNoIdempotencyHeaderWithoutAKey(t *testing.T) {
	var present bool
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, present = r.Header["Idempotency-Key"]
		success(t, w, http.StatusOK, `{"message_id":1}`)
	})
	if _, err := client.Emails.Send(context.Background(), sendRequest()); err != nil {
		t.Fatal(err)
	}
	if present {
		t.Error("Idempotency-Key was sent without a key")
	}
}

func TestIdempotencyOnEverySendEndpoint(t *testing.T) {
	type call struct {
		name string
		run  func(*Client) error
	}
	calls := []call{
		{"Send", func(c *Client) error {
			_, err := c.Emails.Send(context.Background(), sendRequest(), WithIdempotencyKey("k"))
			return err
		}},
		{"SendBatch", func(c *Client) error {
			_, err := c.Emails.SendBatch(context.Background(), []*SendEmailRequest{sendRequest()}, WithIdempotencyKey("k"))
			return err
		}},
		{"SendWithTemplate", func(c *Client) error {
			_, err := c.Emails.SendWithTemplate(context.Background(),
				&SendWithTemplateRequest{SendEmailRequest: *sendRequest(), Template: "welcome"},
				WithIdempotencyKey("k"))
			return err
		}},
		{"SendWithTemplateBatch", func(c *Client) error {
			_, err := c.Emails.SendWithTemplateBatch(context.Background(),
				[]*SendWithTemplateRequest{{SendEmailRequest: *sendRequest(), Template: "welcome"}},
				WithIdempotencyKey("k"))
			return err
		}},
	}
	// The API claims all four send endpoints, so all four have to carry
	// the key.
	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			var gotKey string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotKey = r.Header.Get("Idempotency-Key")
				success(t, w, http.StatusOK, `{"message_id":1,"messages":[]}`)
			})
			if err := c.run(client); err != nil {
				t.Fatal(err)
			}
			if gotKey != "k" {
				t.Errorf("Idempotency-Key = %q", gotKey)
			}
		})
	}
}

func TestReusedKeyForAnotherBodyIsRefused(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusConflict, "InvalidIdempotentRequest",
			"The same idempotency key was used with a different request")
	})
	_, err := client.Emails.Send(context.Background(), sendRequest(), WithIdempotencyKey("reused"))
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "InvalidIdempotentRequest" || apiErr.StatusCode != http.StatusConflict {
		t.Errorf("err = %v", err)
	}
}

func TestSendLimitExceeded(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusTooManyRequests, "SendLimitExceeded", "the send allowance is used up")
	})
	_, err := client.Emails.Send(context.Background(), sendRequest())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "SendLimitExceeded" {
		t.Errorf("err = %v", err)
	}
}

func TestSendToStream(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/streams/newsletter/send" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusAccepted, `{"queued":42,"skipped":3}`)
	})
	result, err := client.Emails.SendToStream(context.Background(), "newsletter", &SendToStreamRequest{
		From:     Address{Email: "news@acme.com"},
		Subject:  "September",
		TextBody: "Hello.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["from"] != "news@acme.com" {
		t.Errorf("body = %v", gotBody)
	}
	if result.Queued != 42 || result.Skipped != 3 {
		t.Errorf("result = %+v", result)
	}
}
