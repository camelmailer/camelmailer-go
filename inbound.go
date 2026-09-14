package camelmailer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// InboundService reads inbound and held messages via
// /api/v2/server/inbound.
//
// It covers mail arriving through an inbound route as well as outbound
// mail the spam filter put on hold, which is why a message here can be
// either retried or released past the hold.
type InboundService struct {
	client *Client
}

// ListInboundOptions filters InboundService.List.
type ListInboundOptions struct {
	ListOptions
	// Status restricts to one delivery status, e.g. "held".
	Status string
	// Stream restricts to one message stream (by permalink).
	Stream string
	// Query is a substring match on subject and addresses.
	Query string
}

// ListInboundResult is one page of inbound and held messages.
type ListInboundResult struct {
	// Inbound is the page of messages.
	Inbound []Message `json:"inbound"`
	// Pagination describes the page window.
	Pagination Pagination `json:"pagination"`
}

// RequeueResult reports what a retry or bypass did.
type RequeueResult struct {
	// Queued is true when the message went back on the delivery queue.
	Queued bool `json:"queued"`
}

// List returns inbound and held messages, newest first. opts may be
// nil.
//
// API: GET /api/v2/server/inbound
func (s *InboundService) List(ctx context.Context, opts *ListInboundOptions) (*ListInboundResult, error) {
	query := url.Values{}
	if opts != nil {
		opts.values(query)
		for key, value := range map[string]string{
			"status": opts.Status,
			"stream": opts.Stream,
			"query":  opts.Query,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
	}
	out := new(ListInboundResult)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/inbound", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one inbound message.
//
// API: GET /api/v2/server/inbound/{id}
func (s *InboundService) Get(ctx context.Context, id int64) (*Message, error) {
	var out struct {
		Message Message `json:"message"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/inbound/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.Message, nil
}

// Retry puts a message back on the delivery queue, for instance after
// fixing the route it should have matched.
//
// API: POST /api/v2/server/inbound/{id}/retry
func (s *InboundService) Retry(ctx context.Context, id int64) (*RequeueResult, error) {
	out := new(RequeueResult)
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/server/inbound/%d/retry", id), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Bypass releases a held message past the hold and delivers it.
//
// API: POST /api/v2/server/inbound/{id}/bypass
func (s *InboundService) Bypass(ctx context.Context, id int64) (*RequeueResult, error) {
	out := new(RequeueResult)
	if err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/v2/server/inbound/%d/bypass", id), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
