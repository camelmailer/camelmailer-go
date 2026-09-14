package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

const campaignJSON = `{"id":7,"name":"September","subject":"What shipped","from":"news@acme.com",` +
	`"status":"draft","total":120,"sent":0,"stream_id":3,` +
	`"stream":{"permalink":"product-news","name":"Product news"},` +
	`"scheduled_at":null,"created_at":"2026-09-01T10:00:00Z","completed_at":null}`

func TestCampaignsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/server/campaigns" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"campaigns":[`+campaignJSON+`]}`)
	})
	campaigns, err := client.Campaigns.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(campaigns) != 1 || campaigns[0].Stream.Permalink != "product-news" {
		t.Errorf("campaigns = %+v", campaigns)
	}
	if campaigns[0].ScheduledAt != nil {
		t.Errorf("scheduled_at = %v, want nil for a draft", campaigns[0].ScheduledAt)
	}
}

func TestCampaignsListForStream(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/streams/product-news/campaigns" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"campaigns":[]}`)
	})
	if _, err := client.Campaigns.ListForStream(context.Background(), "product-news"); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignsGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/campaigns/7" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"campaign":`+campaignJSON+
			`,"stats":{"total":120,"sent":118,"delivered":110,"failed":8,"opened":40,"clicked":9,"unsubscribed":1}}`)
	})
	detail, err := client.Campaigns.Get(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Stats.Delivered != 110 || detail.Campaign.ID != 7 {
		t.Errorf("detail = %+v", detail)
	}
}

func TestCampaignsGetForStream(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/streams/product-news/campaigns/7" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"campaign":`+campaignJSON+`}`)
	})
	if _, err := client.Campaigns.GetForStream(context.Background(), "product-news", 7); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignsCreateDraft(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// The planning route: the stream travels in the body, not the path.
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/campaigns" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"campaign":`+campaignJSON+`}`)
	})
	campaign, err := client.Campaigns.CreateDraft(context.Background(), &CreateDraftCampaignRequest{
		Stream:  "product-news",
		From:    "news@acme.com",
		Name:    "September",
		Subject: "What shipped",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["stream"] != "product-news" {
		t.Errorf("body = %v", gotBody)
	}
	if _, ok := gotBody["scheduled_at"]; ok {
		t.Errorf("scheduled_at should be absent for a draft, body = %v", gotBody)
	}
	if campaign.Status != "draft" {
		t.Errorf("status = %s, want draft", campaign.Status)
	}
}

func TestCampaignsCreateDraftScheduled(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"campaign":{"id":8,"status":"scheduled"}}`)
	})
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	campaign, err := client.Campaigns.CreateDraft(context.Background(), &CreateDraftCampaignRequest{
		Stream:      "product-news",
		From:        "news@acme.com",
		ScheduledAt: &at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["scheduled_at"] != "2026-10-01T08:00:00Z" {
		t.Errorf("scheduled_at = %v", gotBody["scheduled_at"])
	}
	if campaign.Status != "scheduled" {
		t.Errorf("status = %s, want scheduled", campaign.Status)
	}
}

func TestCampaignsCreateAndSend(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		// The stream-scoped route expands to the subscribers before it
		// answers, so the campaign comes back already sending.
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/streams/product-news/campaigns" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusCreated, `{"campaign":{"id":9,"status":"sending"}}`)
	})
	campaign, err := client.Campaigns.CreateAndSend(context.Background(), "product-news", &CreateAndSendCampaignRequest{
		Name: "Status update",
		From: "news@acme.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "sending" {
		t.Errorf("status = %s, want sending", campaign.Status)
	}
}

func TestCampaignsUpdateSchedules(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/server/campaigns/7" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"campaign":{"id":7,"status":"scheduled"}}`)
	})
	at := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	if _, err := client.Campaigns.Update(context.Background(), 7, &UpdateCampaignRequest{ScheduledAt: &at}); err != nil {
		t.Fatal(err)
	}
	if gotBody["scheduled_at"] != "2026-10-01T08:00:00Z" {
		t.Errorf("scheduled_at = %v", gotBody["scheduled_at"])
	}
}

func TestCampaignsUpdateClearsSchedule(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"campaign":{"id":7,"status":"draft"}}`)
	})
	campaign, err := client.Campaigns.Update(context.Background(), 7, &UpdateCampaignRequest{ClearSchedule: true})
	if err != nil {
		t.Fatal(err)
	}
	// An omitted field leaves the schedule standing; only an explicit
	// null clears it, so the key has to be present and null.
	value, present := gotBody["scheduled_at"]
	if !present || value != nil {
		t.Errorf("scheduled_at = %v (present %v), want an explicit null", value, present)
	}
	if campaign.Status != "draft" {
		t.Errorf("status = %s, want draft", campaign.Status)
	}
}

func TestCampaignsUpdateKeepsScheduleWhenUnset(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"campaign":`+campaignJSON+`}`)
	})
	if _, err := client.Campaigns.Update(context.Background(), 7, &UpdateCampaignRequest{Subject: String("Corrected")}); err != nil {
		t.Fatal(err)
	}
	if _, present := gotBody["scheduled_at"]; present {
		t.Errorf("scheduled_at should be absent, body = %v", gotBody)
	}
}

func TestCampaignsSendAndCancel(t *testing.T) {
	var paths []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		success(t, w, http.StatusOK, `{"campaign":{"id":7,"status":"sending"}}`)
	})
	if _, err := client.Campaigns.Send(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Campaigns.Cancel(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/api/v2/server/campaigns/7/send" || paths[1] != "/api/v2/server/campaigns/7/cancel" {
		t.Errorf("paths = %v", paths)
	}
}

func TestCampaignsUpdateSendingIsRefused(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnprocessableEntity, "ValidationError", "a sent campaign can no longer be edited")
	})
	_, err := client.Campaigns.Update(context.Background(), 7, &UpdateCampaignRequest{Subject: String("Too late")})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "ValidationError" {
		t.Errorf("err = %v", err)
	}
}
