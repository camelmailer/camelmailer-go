package camelmailer

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// BouncesService reads bounced messages via /api/v2/server/bounces.
type BouncesService struct {
	client *Client
}

// ListBouncesOptions filters BouncesService.List.
type ListBouncesOptions struct {
	ListOptions
	// Scope restricts to "incoming" or "outgoing" messages.
	Scope string
	// Status restricts to one delivery status.
	Status string
	// Tag restricts to one tag.
	Tag string
	// Query is a substring match on subject and addresses.
	Query string
}

// ListBouncesResult is one page of bounced messages.
type ListBouncesResult struct {
	// Bounces is the page of bounced messages.
	Bounces []Message `json:"bounces"`
	// Pagination describes the page window.
	Pagination Pagination `json:"pagination"`
}

// List returns the server's bounced messages, filtered and paginated.
// opts may be nil.
//
// API: GET /api/v2/server/bounces
func (s *BouncesService) List(ctx context.Context, opts *ListBouncesOptions) (*ListBouncesResult, error) {
	query := url.Values{}
	if opts != nil {
		opts.ListOptions.values(query)
		for key, value := range map[string]string{
			"scope":  opts.Scope,
			"status": opts.Status,
			"tag":    opts.Tag,
			"query":  opts.Query,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
	}
	out := new(ListBouncesResult)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/bounces", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one bounced message by id.
//
// API: GET /api/v2/server/bounces/{id}
func (s *BouncesService) Get(ctx context.Context, id int64) (*Message, error) {
	var out struct {
		Bounce Message `json:"bounce"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/bounces/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out.Bounce, nil
}
