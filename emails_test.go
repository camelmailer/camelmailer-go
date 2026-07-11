package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestEmailsSend(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   map[string]any
	)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{
			"message_id": 100,
			"recipients": [
				{"rcpt_to":"ada@example.com","message_id":100,"token":"tok1","status":"queued"}
			]
		}`)
	})

	sent, err := client.Emails.Send(context.Background(), &SendEmailRequest{
		From:     Address{Email: "billing@acme.com"},
		To:       []Address{{Email: "ada@example.com", Name: "Ada"}},
		Subject:  "Your receipt",
		TextBody: "Thanks!",
		Tag:      "receipt",
		Headers:  map[string]string{"X-Custom": "1"},
		Metadata: map[string]any{"order_id": 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v2/server/messages" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}
	if gotBody["from"] != "billing@acme.com" {
		t.Errorf("from = %v, want bare string", gotBody["from"])
	}
	to := gotBody["to"].([]any)[0].(map[string]any)
	if to["email"] != "ada@example.com" || to["name"] != "Ada" {
		t.Errorf("to = %v", to)
	}
	if gotBody["tag"] != "receipt" {
		t.Errorf("tag = %v", gotBody["tag"])
	}
	if sent.MessageID != 100 {
		t.Errorf("MessageID = %d", sent.MessageID)
	}
	if len(sent.Recipients) != 1 || sent.Recipients[0].Token != "tok1" || sent.Recipients[0].Status != "queued" {
		t.Errorf("recipients = %+v", sent.Recipients)
	}
}

func TestEmailsSendValidationError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnprocessableEntity, "ValidationError",
			`From domain "acme.com" is not a verified sender for this server`)
	})
	_, err := client.Emails.Send(context.Background(), &SendEmailRequest{
		From: Address{Email: "billing@acme.com"},
		To:   []Address{{Email: "ada@example.com"}},
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.Code != "ValidationError" || apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("unexpected error: %+v", apiErr)
	}
}

func TestEmailsSendBatch(t *testing.T) {
	var gotBody []map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/batch" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"messages":[
			{"status":"success","data":{"message_id":1,"recipients":[]}},
			{"status":"error","error":{"code":"ParameterMissing","message":"param is missing or the value is empty: from"}}
		]}`)
	})

	results, err := client.Emails.SendBatch(context.Background(), []*SendEmailRequest{
		{From: Address{Email: "a@acme.com"}, To: []Address{{Email: "x@example.com"}}},
		{To: []Address{{Email: "y@example.com"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(gotBody) != 2 {
		t.Fatalf("batch body has %d entries, want a bare JSON array of 2", len(gotBody))
	}
	if len(results.Messages) != 2 {
		t.Fatalf("results = %+v", results)
	}
	if results.Messages[0].Status != "success" || results.Messages[0].Data.MessageID != 1 {
		t.Errorf("first result = %+v", results.Messages[0])
	}
	if results.Messages[1].Status != "error" || results.Messages[1].Error.Code != "ParameterMissing" {
		t.Errorf("second result = %+v", results.Messages[1])
	}
}

func TestEmailsSendWithTemplate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/with_template" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"message_id":5,"recipients":[{"rcpt_to":"ada@example.com","message_id":5,"token":"t","status":"queued"}]}`)
	})

	sent, err := client.Emails.SendWithTemplate(context.Background(), &SendWithTemplateRequest{
		SendEmailRequest: SendEmailRequest{
			From: Address{Email: "hello@acme.com"},
			To:   []Address{{Email: "ada@example.com"}},
		},
		Template:      "welcome",
		TemplateModel: map[string]any{"name": "Ada"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// template fields must be flattened next to from/to, not nested
	if gotBody["template"] != "welcome" {
		t.Errorf("template = %v (body: %v)", gotBody["template"], gotBody)
	}
	if gotBody["from"] != "hello@acme.com" {
		t.Errorf("from = %v, want flattened send fields", gotBody["from"])
	}
	model := gotBody["template_model"].(map[string]any)
	if model["name"] != "Ada" {
		t.Errorf("template_model = %v", model)
	}
	if sent.MessageID != 5 {
		t.Errorf("MessageID = %d", sent.MessageID)
	}
}

func TestEmailsSendWithTemplateBatch(t *testing.T) {
	var gotBody []map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/with_template/batch" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"messages":[{"status":"success","data":{"message_id":9,"recipients":[]}}]}`)
	})

	results, err := client.Emails.SendWithTemplateBatch(context.Background(), []*SendWithTemplateRequest{
		{
			SendEmailRequest: SendEmailRequest{
				From: Address{Email: "hello@acme.com"},
				To:   []Address{{Email: "ada@example.com"}},
			},
			Template: "welcome",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(gotBody) != 1 || gotBody[0]["template"] != "welcome" {
		t.Errorf("body = %v", gotBody)
	}
	if len(results.Messages) != 1 || results.Messages[0].Data.MessageID != 9 {
		t.Errorf("results = %+v", results)
	}
}

func TestEmailsGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/42" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{
			"message": {
				"id": 42, "token": "tok", "scope": "outgoing",
				"rcpt_to": "ada@example.com", "mail_from": "billing@acme.com",
				"subject": "Hi", "message_id": "<abc@acme.com>", "tag": "receipt",
				"status": "Sent", "bounce": false, "spam_status": null,
				"spam_score": 0.1, "held": false, "threat": false, "size": 1234,
				"metadata": {"order_id": 7}, "stream_id": 1, "bypassed": false,
				"created_at": "2026-07-11T10:00:00+00:00"
			},
			"deliveries": [
				{"id":1,"status":"Sent","details":"250 OK","output":"accepted","sent_with_ssl":true,"created_at":"2026-07-11T10:00:01+00:00"}
			]
		}`)
	})

	email, err := client.Emails.Get(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if email.Message.ID != 42 || email.Message.Status != "Sent" || email.Message.RcptTo != "ada@example.com" {
		t.Errorf("message = %+v", email.Message)
	}
	if len(email.Deliveries) != 1 || !email.Deliveries[0].SentWithSSL {
		t.Errorf("deliveries = %+v", email.Deliveries)
	}
}

func TestEmailsGetNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotFound", "Resource not found")
	})
	_, err := client.Emails.Get(context.Background(), 999)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotFound" {
		t.Fatalf("err = %v", err)
	}
}

func TestEmailsList(t *testing.T) {
	var gotQuery map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages" || r.Method != http.MethodGet {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		gotQuery = r.URL.Query()
		success(t, w, http.StatusOK, `{
			"messages": [{"id":1,"token":"a","scope":"outgoing","rcpt_to":"x@example.com","bounce":false,"held":false,"threat":false,"bypassed":false,"created_at":"2026-07-11T10:00:00+00:00"}],
			"pagination": {"page":2,"per_page":10,"total":11,"total_pages":2}
		}`)
	})

	list, err := client.Emails.List(context.Background(), &ListEmailsOptions{
		ListOptions: ListOptions{Page: 2, PerPage: 10},
		Scope:       "outgoing",
		Status:      "Sent",
		Tag:         "receipt",
		Query:       "ada",
		Stream:      "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"page": "2", "per_page": "10", "scope": "outgoing",
		"status": "Sent", "tag": "receipt", "query": "ada", "stream": "default",
	} {
		if got := gotQuery[key]; len(got) != 1 || got[0] != want {
			t.Errorf("query %s = %v, want %s", key, got, want)
		}
	}
	if len(list.Messages) != 1 || list.Messages[0].ID != 1 {
		t.Errorf("messages = %+v", list.Messages)
	}
	if list.Pagination.Page != 2 || list.Pagination.Total != 11 {
		t.Errorf("pagination = %+v", list.Pagination)
	}
}

func TestEmailsListNilOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %s, want empty", r.URL.RawQuery)
		}
		success(t, w, http.StatusOK, `{"messages":[],"pagination":{"page":1,"per_page":25,"total":0,"total_pages":0}}`)
	})
	list, err := client.Emails.List(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Messages) != 0 {
		t.Errorf("messages = %+v", list.Messages)
	}
}

func TestEmailsDeliveries(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/7/deliveries" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"deliveries":[{"id":3,"status":"SoftFail","details":"451 try later","output":"","sent_with_ssl":false,"created_at":"2026-07-11T10:00:00+00:00"}]}`)
	})
	deliveries, err := client.Emails.Deliveries(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].Status != "SoftFail" {
		t.Errorf("deliveries = %+v", deliveries)
	}
}

func TestEmailsOpens(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/7/opens" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"opens":[{"ip_address":"203.0.113.9","user_agent":"Mozilla/5.0","url":null,"created_at":"2026-07-11T10:00:00+00:00"}]}`)
	})
	opens, err := client.Emails.Opens(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(opens) != 1 || opens[0].IPAddress != "203.0.113.9" {
		t.Errorf("opens = %+v", opens)
	}
}

func TestEmailsClicks(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/7/clicks" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"clicks":[{"ip_address":"203.0.113.9","user_agent":"Mozilla/5.0","url":"https://acme.com","created_at":"2026-07-11T10:00:00+00:00"}]}`)
	})
	clicks, err := client.Emails.Clicks(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(clicks) != 1 || clicks[0].URL != "https://acme.com" {
		t.Errorf("clicks = %+v", clicks)
	}
}

func TestEmailsRaw(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/messages/7/raw" {
			t.Errorf("path = %s", r.URL.Path)
		}
		// base64("Subject: Hi\r\n\r\nHello")
		success(t, w, http.StatusOK, `{"raw_message":"U3ViamVjdDogSGkNCg0KSGVsbG8="}`)
	})
	raw, err := client.Emails.Raw(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "Subject: Hi\r\n\r\nHello" {
		t.Errorf("raw = %q", raw)
	}
}

func TestEmailsRawPrivacyMode(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotAvailable", "Raw message content is not retained in privacy mode")
	})
	_, err := client.Emails.Raw(context.Background(), 7)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotAvailable" {
		t.Fatalf("err = %v", err)
	}
}
