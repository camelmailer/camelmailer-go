package camelmailer

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// LogsService reads the server's own request log and tag index via
// /api/v2/server/logs and /api/v2/server/tags.
//
// Useful when a send did not arrive and the question is whether the
// request ever reached the API, and with what answer.
type LogsService struct {
	client *Client
}

// APIRequest is one logged API request.
type APIRequest struct {
	// ID is the numeric log id.
	ID int64 `json:"id"`
	// Method is the HTTP method.
	Method string `json:"method"`
	// Path is the request path.
	Path string `json:"path"`
	// StatusCode is the HTTP status the API answered with.
	StatusCode int `json:"status_code"`
	// DurationMs is how long the request took, in milliseconds.
	DurationMs int64 `json:"duration_ms"`
	// UserAgent is the client's User-Agent header.
	UserAgent string `json:"user_agent"`
	// CreatedAt is when the request arrived.
	CreatedAt *time.Time `json:"created_at"`
}

// ListLogsOptions filters LogsService.List.
type ListLogsOptions struct {
	ListOptions
	// Method restricts to one HTTP method.
	Method string
	// Path is a substring match on the request path.
	Path string
	// Status restricts to one HTTP status code.
	Status string
}

// ListLogsResult is one page of logged requests.
type ListLogsResult struct {
	// Requests is the page of logged requests.
	Requests []APIRequest `json:"requests"`
	// Pagination describes the page window.
	Pagination Pagination `json:"pagination"`
}

// TagCount is one tag with how often the server's recent messages used
// it.
type TagCount struct {
	// Tag is the tag itself.
	Tag string `json:"tag"`
	// Count is how many messages carry it.
	Count int64 `json:"count"`
}

// List returns logged API requests, newest first. opts may be nil.
//
// API: GET /api/v2/server/logs
func (s *LogsService) List(ctx context.Context, opts *ListLogsOptions) (*ListLogsResult, error) {
	query := url.Values{}
	if opts != nil {
		opts.values(query)
		for key, value := range map[string]string{
			"method": opts.Method,
			"path":   opts.Path,
			"status": opts.Status,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
	}
	out := new(ListLogsResult)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/logs", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Tags returns the tags used by the server's recent messages, most used
// first.
//
// API: GET /api/v2/server/tags
func (s *LogsService) Tags(ctx context.Context) ([]TagCount, error) {
	var out struct {
		Tags []TagCount `json:"tags"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/tags", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Tags, nil
}
