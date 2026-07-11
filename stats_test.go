package camelmailer

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestStatsGet(t *testing.T) {
	var gotQuery map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/stats" {
			t.Errorf("path = %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		success(t, w, http.StatusOK, `{"stats":{
			"total":10,"incoming":2,"outgoing":8,"sent":7,"pending":1,"held":0,
			"bounced":0,"soft_fail":0,"hard_fail":0,
			"opens":5,"clicks":3,"unique_opens":4,"unique_clicks":2
		}}`)
	})

	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	stats, err := client.Stats.Get(context.Background(), &StatsOptions{From: from, To: to})
	if err != nil {
		t.Fatal(err)
	}
	if got := gotQuery["from"]; len(got) != 1 || got[0] != "2026-07-01T00:00:00Z" {
		t.Errorf("from = %v", got)
	}
	if got := gotQuery["to"]; len(got) != 1 || got[0] != "2026-07-11T00:00:00Z" {
		t.Errorf("to = %v", got)
	}
	if stats.Total != 10 || stats.Sent != 7 || stats.UniqueOpens != 4 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestStatsGetNilOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("query = %s, want empty", r.URL.RawQuery)
		}
		success(t, w, http.StatusOK, `{"stats":{"total":0}}`)
	})
	if _, err := client.Stats.Get(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestStatsDeliveries(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/stats/deliveries" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"queued":12,"domains":[{"domain":"example.com","queued":9},{"domain":"acme.com","queued":3}]}`)
	})
	stats, err := client.Stats.Deliveries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Queued != 12 || len(stats.Domains) != 2 || stats.Domains[0].Domain != "example.com" || stats.Domains[0].Queued != 9 {
		t.Errorf("stats = %+v", stats)
	}
}

func TestStatsUnauthorized(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnauthorized, "Unauthorized", "This server has been suspended")
	})
	_, err := client.Stats.Get(context.Background(), nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "Unauthorized" {
		t.Fatalf("err = %v", err)
	}
}
