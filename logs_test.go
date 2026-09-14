package camelmailer

import (
	"context"
	"net/http"
	"testing"
)

func TestLogsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/logs" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("per_page"); got != "25" {
			t.Errorf("per_page = %s", got)
		}
		success(t, w, http.StatusOK,
			`{"requests":[{"id":1,"method":"POST","path":"/api/v2/server/messages",`+
				`"status_code":201,"duration_ms":12,"user_agent":"camelmailer-go/0.2.0",`+
				`"created_at":"2026-09-14T08:00:00Z"}],`+
				`"pagination":{"page":1,"per_page":25,"total":1,"total_pages":1}}`)
	})
	result, err := client.Logs.List(context.Background(), &ListLogsOptions{
		ListOptions: ListOptions{PerPage: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Requests) != 1 || result.Requests[0].StatusCode != 201 {
		t.Errorf("result = %+v", result)
	}
	if result.Requests[0].DurationMs != 12 {
		t.Errorf("duration = %d", result.Requests[0].DurationMs)
	}
}

func TestLogsTags(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/tags" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"tags":[{"tag":"receipt","count":12},{"tag":"signup","count":4}]}`)
	})
	tags, err := client.Logs.Tags(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 || tags[0].Tag != "receipt" || tags[0].Count != 12 {
		t.Errorf("tags = %+v", tags)
	}
}
