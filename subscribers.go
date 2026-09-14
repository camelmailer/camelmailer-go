package camelmailer

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// SubscribersService manages the opt-in audience of a broadcast stream
// via /api/v2/server/streams/{permalink}/subscribers.
//
// A broadcast send to an address that is not subscribed is refused, so
// this list is the audience.
type SubscribersService struct {
	client *Client
}

// Subscriber is one address on a broadcast stream.
type Subscriber struct {
	// ID is the numeric subscriber id.
	ID int64 `json:"id"`
	// Address is the email address.
	Address string `json:"address"`
	// Status is "subscribed" or "unsubscribed".
	Status string `json:"status"`
	// CreatedAt is when the subscription row was created.
	CreatedAt *time.Time `json:"created_at"`
}

// AddSubscriberRequest is the payload for SubscribersService.Add. It
// upserts by address, so calling it twice is safe.
type AddSubscriberRequest struct {
	// Address is the email address (required).
	Address string `json:"address"`
	// Status sets "subscribed" (default) or "unsubscribed".
	Status string `json:"status,omitempty"`
}

// ImportSubscribersResult reports what an import wrote. Blanks and
// duplicates within the request are skipped, so Added can be lower than
// the number of addresses you passed.
type ImportSubscribersResult struct {
	// Added counts the subscriptions written.
	Added int64 `json:"added"`
	// Total counts the stream's subscribers after the import.
	Total int64 `json:"total"`
}

// subscriberData is the {"subscriber": …} response wrapper.
type subscriberData struct {
	Subscriber Subscriber `json:"subscriber"`
}

// subscribersPath builds the collection path for one stream.
func subscribersPath(permalink string) string {
	return "/api/v2/server/streams/" + url.PathEscape(permalink) + "/subscribers"
}

// List returns the stream's subscribers, subscribed and unsubscribed
// alike.
//
// API: GET /api/v2/server/streams/{permalink}/subscribers
func (s *SubscribersService) List(ctx context.Context, permalink string) ([]Subscriber, error) {
	var out struct {
		Subscribers []Subscriber `json:"subscribers"`
	}
	if err := s.client.do(ctx, http.MethodGet, subscribersPath(permalink), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Subscribers, nil
}

// Add adds or updates one subscriber.
//
// API: POST /api/v2/server/streams/{permalink}/subscribers
func (s *SubscribersService) Add(ctx context.Context, permalink string, req *AddSubscriberRequest) (*Subscriber, error) {
	out := new(subscriberData)
	if err := s.client.do(ctx, http.MethodPost, subscribersPath(permalink), nil, req, out); err != nil {
		return nil, err
	}
	return &out.Subscriber, nil
}

// Import adds many addresses at once, all as subscribed.
//
// API: POST /api/v2/server/streams/{permalink}/subscribers/import
func (s *SubscribersService) Import(ctx context.Context, permalink string, addresses []string) (*ImportSubscribersResult, error) {
	body := struct {
		Addresses []string `json:"addresses"`
	}{Addresses: addresses}
	out := new(ImportSubscribersResult)
	if err := s.client.do(ctx, http.MethodPost, subscribersPath(permalink)+"/import", nil, body, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Complaint records a spam complaint against an address: it writes a
// stream-scoped suppression and flips the subscription to
// "unsubscribed". Idempotent, so a feedback loop can replay it safely.
//
// API: POST /api/v2/server/streams/{permalink}/subscribers/{address}/complaint
func (s *SubscribersService) Complaint(ctx context.Context, permalink, address string) (*Subscriber, error) {
	out := new(subscriberData)
	path := subscribersPath(permalink) + "/" + url.PathEscape(address) + "/complaint"
	if err := s.client.do(ctx, http.MethodPost, path, nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Subscriber, nil
}

// Remove deletes a subscriber from the stream entirely.
//
// API: DELETE /api/v2/server/streams/{permalink}/subscribers/{address}
func (s *SubscribersService) Remove(ctx context.Context, permalink, address string) error {
	path := subscribersPath(permalink) + "/" + url.PathEscape(address)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil, nil)
}
