package camelmailer

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestBouncesList(t *testing.T) {
	var gotQuery map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/server/bounces" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		gotQuery = r.URL.Query()
		success(t, w, http.StatusOK, `{
			"bounces": [{"id":9,"token":"b","scope":"incoming","rcpt_to":"x@example.com","bounce":true,"held":false,"threat":false,"bypassed":false,"created_at":"2026-07-11T10:00:00+00:00"}],
			"pagination": {"page":1,"per_page":25,"total":1,"total_pages":1}
		}`)
	})

	list, err := client.Bounces.List(context.Background(), &ListBouncesOptions{
		ListOptions: ListOptions{Page: 1, PerPage: 25},
		Query:       "example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotQuery["query"]; len(got) != 1 || got[0] != "example.com" {
		t.Errorf("query = %v", got)
	}
	if len(list.Bounces) != 1 || !list.Bounces[0].Bounce {
		t.Errorf("bounces = %+v", list.Bounces)
	}
	if list.Pagination.Total != 1 {
		t.Errorf("pagination = %+v", list.Pagination)
	}
}

func TestBouncesGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/bounces/9" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"bounce":{"id":9,"token":"b","scope":"incoming","rcpt_to":"x@example.com","bounce":true,"held":false,"threat":false,"bypassed":false,"created_at":"2026-07-11T10:00:00+00:00"}}`)
	})
	bounce, err := client.Bounces.Get(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if bounce.ID != 9 || !bounce.Bounce {
		t.Errorf("bounce = %+v", bounce)
	}
}

func TestBouncesGetNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotFound", "Resource not found")
	})
	_, err := client.Bounces.Get(context.Background(), 404)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotFound" {
		t.Fatalf("err = %v", err)
	}
}
