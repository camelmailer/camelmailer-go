package camelmailer

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// StatsService reads message and delivery-queue counters via
// /api/v2/server/stats.
type StatsService struct {
	client *Client
}

// Stats are aggregate message and engagement counters of the server.
type Stats struct {
	// Total is the total number of messages.
	Total int64 `json:"total"`
	// Incoming counts incoming messages.
	Incoming int64 `json:"incoming"`
	// Outgoing counts outgoing messages.
	Outgoing int64 `json:"outgoing"`
	// Sent counts successfully sent messages.
	Sent int64 `json:"sent"`
	// Pending counts messages waiting in the queue.
	Pending int64 `json:"pending"`
	// Held counts held messages.
	Held int64 `json:"held"`
	// Bounced counts bounced messages.
	Bounced int64 `json:"bounced"`
	// SoftFail counts soft delivery failures.
	SoftFail int64 `json:"soft_fail"`
	// HardFail counts hard delivery failures.
	HardFail int64 `json:"hard_fail"`
	// Opens counts open events.
	Opens int64 `json:"opens"`
	// Clicks counts click events.
	Clicks int64 `json:"clicks"`
	// UniqueOpens counts messages opened at least once.
	UniqueOpens int64 `json:"unique_opens"`
	// UniqueClicks counts messages clicked at least once.
	UniqueClicks int64 `json:"unique_clicks"`
}

// StatsOptions restricts StatsService.Get to a created-at window.
// Zero times are omitted.
type StatsOptions struct {
	// From is the inclusive window start.
	From time.Time
	// To is the inclusive window end.
	To time.Time
}

// DomainQueue is the queued-message count for one recipient domain.
type DomainQueue struct {
	// Domain is the recipient domain.
	Domain string `json:"domain"`
	// Queued is the number of queued messages for the domain.
	Queued int64 `json:"queued"`
}

// DeliveryStats describes the outbound delivery queue.
type DeliveryStats struct {
	// Queued is the total number of queued messages.
	Queued int64 `json:"queued"`
	// Domains breaks the queue down by recipient domain.
	Domains []DomainQueue `json:"domains"`
}

// Get returns the server's message counters, optionally restricted to
// a time window. opts may be nil.
//
// API: GET /api/v2/server/stats
func (s *StatsService) Get(ctx context.Context, opts *StatsOptions) (*Stats, error) {
	query := url.Values{}
	if opts != nil {
		setTime(query, "from", opts.From)
		setTime(query, "to", opts.To)
	}
	var out struct {
		Stats Stats `json:"stats"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/stats", query, nil, &out); err != nil {
		return nil, err
	}
	return &out.Stats, nil
}

// Deliveries returns the current outbound queue depth, broken down by
// recipient domain.
//
// API: GET /api/v2/server/stats/deliveries
func (s *StatsService) Deliveries(ctx context.Context) (*DeliveryStats, error) {
	out := new(DeliveryStats)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/stats/deliveries", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
