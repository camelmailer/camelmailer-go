package camelmailer

import (
	"context"
	"net/http"
	"testing"
)

func TestInboundList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/inbound" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("status"); got != "held" {
			t.Errorf("status = %s", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "50" {
			t.Errorf("per_page = %s", got)
		}
		// The page comes back under "inbound", not "messages".
		success(t, w, http.StatusOK,
			`{"inbound":[{"id":55,"status":"Held","held":true,"rcpt_to":"support@acme.com"}],`+
				`"pagination":{"page":1,"per_page":50,"total":1,"total_pages":1}}`)
	})
	result, err := client.Inbound.List(context.Background(), &ListInboundOptions{
		ListOptions: ListOptions{PerPage: 50},
		Status:      "held",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Inbound) != 1 || !result.Inbound[0].Held {
		t.Errorf("result = %+v", result)
	}
	if result.Pagination.Total != 1 {
		t.Errorf("pagination = %+v", result.Pagination)
	}
}

func TestInboundListWithoutOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %s, want none", r.URL.RawQuery)
		}
		success(t, w, http.StatusOK, `{"inbound":[],"pagination":{}}`)
	})
	if _, err := client.Inbound.List(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInboundGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/inbound/55" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"message":{"id":55,"status":"Held","held":true}}`)
	})
	message, err := client.Inbound.Get(context.Background(), 55)
	if err != nil {
		t.Fatal(err)
	}
	if message.ID != 55 || !message.Held {
		t.Errorf("message = %+v", message)
	}
}

func TestInboundRetryAndBypass(t *testing.T) {
	var paths []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		// The API names this "requeued", and answers with the message too.
		success(t, w, http.StatusOK, `{"requeued":true,"message":{"id":55,"status":"Pending"}}`)
	})
	ctx := context.Background()
	retried, err := client.Inbound.Retry(ctx, 55)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Inbound.Bypass(ctx, 55); err != nil {
		t.Fatal(err)
	}
	if !retried.Requeued {
		t.Errorf("retry = %+v", retried)
	}
	if retried.Message.ID != 55 {
		t.Errorf("message = %+v", retried.Message)
	}
	if paths[0] != "POST /api/v2/server/inbound/55/retry" || paths[1] != "POST /api/v2/server/inbound/55/bypass" {
		t.Errorf("paths = %v", paths)
	}
}
