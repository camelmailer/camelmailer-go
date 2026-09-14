package camelmailer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// CampaignsService plans and sends broadcast campaigns via
// /api/v2/server/campaigns.
//
// A campaign is content plus an audience. There are two ways to create
// one and they behave differently: CreateDraft writes it and waits,
// while CreateAndSend expands it to the stream's subscribers before the
// call returns.
type CampaignsService struct {
	client *Client
}

// CampaignStream names the audience stream carried on every campaign,
// so a list needs no second lookup.
type CampaignStream struct {
	// Permalink identifies the stream in API calls.
	Permalink string `json:"permalink"`
	// Name is the stream's display name.
	Name string `json:"name"`
}

// Campaign is one broadcast campaign.
type Campaign struct {
	// ID is the numeric campaign id.
	ID int64 `json:"id"`
	// Name is the display name.
	Name string `json:"name"`
	// Subject is the message subject.
	Subject string `json:"subject"`
	// From is the sender address.
	From string `json:"from"`
	// HTMLBody is the HTML part.
	HTMLBody string `json:"html_body"`
	// TextBody is the plain-text part.
	TextBody string `json:"text_body"`
	// Status is the lifecycle state: "draft", "scheduled", "sending",
	// "sent", "failed" or "canceled". Only draft and scheduled are
	// editable.
	Status string `json:"status"`
	// Total is the recipient count snapshotted when the send begins.
	Total int64 `json:"total"`
	// Sent counts the recipients expanded into messages so far.
	Sent int64 `json:"sent"`
	// StreamID is the id of the audience stream.
	StreamID int64 `json:"stream_id"`
	// Stream carries the audience stream's permalink and name.
	Stream CampaignStream `json:"stream"`
	// ScheduledAt is the send time of a scheduled campaign.
	ScheduledAt *time.Time `json:"scheduled_at"`
	// CreatedAt is when the campaign row was created.
	CreatedAt *time.Time `json:"created_at"`
	// CompletedAt is when expansion finished (sent or failed).
	CompletedAt *time.Time `json:"completed_at"`
}

// CampaignStats are the per-campaign counters, attributed through the
// messages the campaign produced.
type CampaignStats struct {
	// Total is the recipient count of the campaign.
	Total int64 `json:"total"`
	// Sent counts the messages created.
	Sent int64 `json:"sent"`
	// Delivered counts the messages delivered.
	Delivered int64 `json:"delivered"`
	// Failed counts the messages that failed.
	Failed int64 `json:"failed"`
	// Opened counts the messages opened at least once.
	Opened int64 `json:"opened"`
	// Clicked counts the messages with at least one click.
	Clicked int64 `json:"clicked"`
	// Unsubscribed counts the resulting unsubscribes.
	Unsubscribed int64 `json:"unsubscribed"`
}

// CampaignDetail is one campaign together with its statistics.
type CampaignDetail struct {
	// Campaign is the campaign itself.
	Campaign Campaign `json:"campaign"`
	// Stats are its counters.
	Stats CampaignStats `json:"stats"`
}

// CreateDraftCampaignRequest is the payload for
// CampaignsService.CreateDraft. The initial status follows what you
// pass: SendNow wins, then a ScheduledAt (status "scheduled"), else a
// draft.
type CreateDraftCampaignRequest struct {
	// Stream is the permalink of the broadcast stream to send to
	// (required).
	Stream string `json:"stream"`
	// From is the bare sender address (required); the broadcast path
	// authorizes its domain.
	From string `json:"from"`
	// Name is the display name.
	Name string `json:"name,omitempty"`
	// Subject is the message subject.
	Subject string `json:"subject,omitempty"`
	// HTMLBody is the HTML part.
	HTMLBody string `json:"html_body,omitempty"`
	// TextBody is the plain-text part.
	TextBody string `json:"text_body,omitempty"`
	// ScheduledAt arms the campaign as "scheduled".
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	// SendNow sends on creation, overriding ScheduledAt.
	SendNow bool `json:"send_now,omitempty"`
}

// CreateAndSendCampaignRequest is the payload for
// CampaignsService.CreateAndSend. There is no schedule here: the send
// starts before the call returns.
type CreateAndSendCampaignRequest struct {
	// Name is the display name (required).
	Name string `json:"name"`
	// From is the bare sender address.
	From string `json:"from,omitempty"`
	// Subject is the message subject.
	Subject string `json:"subject,omitempty"`
	// HTMLBody is the HTML part.
	HTMLBody string `json:"html_body,omitempty"`
	// TextBody is the plain-text part.
	TextBody string `json:"text_body,omitempty"`
}

// UpdateCampaignRequest changes the given fields of a draft or
// scheduled campaign. Nil fields are left unchanged; use String for the
// pointer fields.
type UpdateCampaignRequest struct {
	// Name replaces the display name.
	Name *string `json:"name,omitempty"`
	// From replaces the sender address.
	From *string `json:"from,omitempty"`
	// Subject replaces the subject.
	Subject *string `json:"subject,omitempty"`
	// HTMLBody replaces the HTML part.
	HTMLBody *string `json:"html_body,omitempty"`
	// TextBody replaces the plain-text part.
	TextBody *string `json:"text_body,omitempty"`
	// ScheduledAt moves a draft to "scheduled". Leave it nil to keep the
	// current schedule.
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	// ClearSchedule sends scheduled_at as an explicit null, which drops a
	// scheduled campaign back to "draft". Set this instead of
	// ScheduledAt: omitting the field and sending null mean different
	// things to the API, and an omitted field leaves the schedule
	// standing.
	ClearSchedule bool `json:"-"`
}

// MarshalJSON implements json.Marshaler so ClearSchedule reaches the
// API as an explicit null rather than as an omitted field.
func (r UpdateCampaignRequest) MarshalJSON() ([]byte, error) {
	type plain UpdateCampaignRequest
	if !r.ClearSchedule {
		return json.Marshal(plain(r))
	}
	r.ScheduledAt = nil
	return json.Marshal(struct {
		plain
		ScheduledAt *time.Time `json:"scheduled_at"`
	}{plain: plain(r)})
}

// campaignData is the {"campaign": …} response wrapper.
type campaignData struct {
	Campaign Campaign `json:"campaign"`
}

// List returns every campaign of the server, newest first.
//
// API: GET /api/v2/server/campaigns
func (s *CampaignsService) List(ctx context.Context) ([]Campaign, error) {
	var out struct {
		Campaigns []Campaign `json:"campaigns"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/campaigns", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Campaigns, nil
}

// ListForStream returns the campaigns of one broadcast stream.
//
// API: GET /api/v2/server/streams/{permalink}/campaigns
func (s *CampaignsService) ListForStream(ctx context.Context, permalink string) ([]Campaign, error) {
	var out struct {
		Campaigns []Campaign `json:"campaigns"`
	}
	path := "/api/v2/server/streams/" + url.PathEscape(permalink) + "/campaigns"
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Campaigns, nil
}

// Get returns one campaign together with its statistics.
//
// API: GET /api/v2/server/campaigns/{id}
func (s *CampaignsService) Get(ctx context.Context, id int64) (*CampaignDetail, error) {
	out := new(CampaignDetail)
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/campaigns/%d", id), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetForStream returns one campaign through its stream.
//
// API: GET /api/v2/server/streams/{permalink}/campaigns/{id}
func (s *CampaignsService) GetForStream(ctx context.Context, permalink string, id int64) (*CampaignDetail, error) {
	out := new(CampaignDetail)
	path := fmt.Sprintf("/api/v2/server/streams/%s/campaigns/%d", url.PathEscape(permalink), id)
	if err := s.client.do(ctx, http.MethodGet, path, nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateDraft creates a campaign without sending it. Name the audience
// with req.Stream; leave req.ScheduledAt nil for a draft, set it for a
// scheduled send, or set req.SendNow to send on creation.
//
// API: POST /api/v2/server/campaigns
func (s *CampaignsService) CreateDraft(ctx context.Context, req *CreateDraftCampaignRequest) (*Campaign, error) {
	out := new(campaignData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/campaigns", nil, req, out); err != nil {
		return nil, err
	}
	return &out.Campaign, nil
}

// CreateAndSend creates a campaign on a broadcast stream and sends it
// immediately. The send starts before this call returns, so there is no
// draft to review and no schedule to set; use CreateDraft when the
// campaign should wait.
//
// API: POST /api/v2/server/streams/{permalink}/campaigns
func (s *CampaignsService) CreateAndSend(ctx context.Context, permalink string, req *CreateAndSendCampaignRequest) (*Campaign, error) {
	out := new(campaignData)
	path := "/api/v2/server/streams/" + url.PathEscape(permalink) + "/campaigns"
	if err := s.client.do(ctx, http.MethodPost, path, nil, req, out); err != nil {
		return nil, err
	}
	return &out.Campaign, nil
}

// Update changes a draft or scheduled campaign. A campaign that is
// already sending cannot be edited and the API answers
// ValidationError.
//
// API: PATCH /api/v2/server/campaigns/{id}
func (s *CampaignsService) Update(ctx context.Context, id int64, req *UpdateCampaignRequest) (*Campaign, error) {
	out := new(campaignData)
	if err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v2/server/campaigns/%d", id), nil, req, out); err != nil {
		return nil, err
	}
	return &out.Campaign, nil
}

// Send sends a campaign now, whatever its schedule said.
//
// API: POST /api/v2/server/campaigns/{id}/send
func (s *CampaignsService) Send(ctx context.Context, id int64) (*Campaign, error) {
	out := new(campaignData)
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/server/campaigns/%d/send", id), nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Campaign, nil
}

// Cancel calls off a scheduled or in-flight campaign. Messages already
// queued are not recalled.
//
// API: POST /api/v2/server/campaigns/{id}/cancel
func (s *CampaignsService) Cancel(ctx context.Context, id int64) (*Campaign, error) {
	out := new(campaignData)
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/server/campaigns/%d/cancel", id), nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Campaign, nil
}
