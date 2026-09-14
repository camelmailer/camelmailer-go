package camelmailer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestSubscribersList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/server/streams/product-news/subscribers" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"subscribers":[{"id":1,"address":"ada@example.com","status":"subscribed","created_at":"2026-09-01T10:00:00Z"}]}`)
	})
	subscribers, err := client.Subscribers.List(context.Background(), "product-news")
	if err != nil {
		t.Fatal(err)
	}
	if len(subscribers) != 1 || subscribers[0].Status != "subscribed" {
		t.Errorf("subscribers = %+v", subscribers)
	}
}

func TestSubscribersAdd(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"subscriber":{"id":1,"address":"ada@example.com","status":"subscribed"}}`)
	})
	subscriber, err := client.Subscribers.Add(context.Background(), "product-news", &AddSubscriberRequest{
		Address: "ada@example.com",
		Name:    "Ada",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["address"] != "ada@example.com" || gotBody["name"] != "Ada" {
		t.Errorf("body = %v", gotBody)
	}
	if subscriber.Address != "ada@example.com" {
		t.Errorf("subscriber = %+v", subscriber)
	}
}

func TestSubscribersImport(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/streams/product-news/subscribers/import" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"added":2,"total":2}`)
	})
	result, err := client.Subscribers.Import(context.Background(), "product-news",
		[]string{"ada@example.com", "grace@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	addresses, _ := gotBody["addresses"].([]any)
	if len(addresses) != 2 {
		t.Errorf("addresses = %v", gotBody["addresses"])
	}
	if result.Added != 2 {
		t.Errorf("result = %+v", result)
	}
}

func TestSubscribersRemoveEscapesTheAddress(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s", r.Method)
		}
		// r.URL.Path is already decoded; EscapedPath shows what went over
		// the wire, and the plus has to survive it or a different address
		// is removed.
		gotPath = r.URL.EscapedPath()
		success(t, w, http.StatusOK, `{"deleted":true}`)
	})
	if err := client.Subscribers.Remove(context.Background(), "product-news", "ada+news@example.com"); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v2/server/streams/product-news/subscribers/ada+news@example.com" {
		t.Errorf("path = %s", gotPath)
	}
}

func TestSubscribersComplaint(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost ||
			r.URL.Path != "/api/v2/server/streams/product-news/subscribers/ada@example.com/complaint" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"subscriber":{"id":1,"address":"ada@example.com","status":"unsubscribed"}}`)
	})
	subscriber, err := client.Subscribers.Complaint(context.Background(), "product-news", "ada@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if subscriber.Status != "unsubscribed" {
		t.Errorf("status = %s", subscriber.Status)
	}
}
