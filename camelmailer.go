// Package camelmailer is the official Go SDK for the CamelMailer
// transactional email platform (https://camelmailer.com).
//
// It covers the Messaging API (/api/v2/server), authenticated with a
// server API key:
//
//	client := camelmailer.NewClient("cm_xxxx")
//	sent, err := client.Emails.Send(ctx, &camelmailer.SendEmailRequest{
//		From:     camelmailer.Address{Email: "billing@acme.com"},
//		To:       []camelmailer.Address{{Email: "ada@example.com"}},
//		Subject:  "Your receipt",
//		TextBody: "Thanks for your purchase.",
//	})
//
// Self-hosted instances configure their own base URL with
// [WithBaseURL]. API failures are returned as [*APIError] and can be
// inspected with errors.As.
package camelmailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the CamelMailer cloud instance. Self-hosted users
// override it with WithBaseURL.
const DefaultBaseURL = "https://app.camelmailer.com"

// Version is the SDK version, sent in the User-Agent header.
const Version = "0.2.0"

const defaultUserAgent = "camelmailer-go/" + Version

// Client talks to one CamelMailer server through the Messaging API.
// Create it with NewClient; the zero value is not usable.
type Client struct {
	apiKey     string
	baseURL    string
	userAgent  string
	httpClient *http.Client

	// Emails sends and reads messages.
	Emails *EmailsService
	// Templates manages stored message templates.
	Templates *TemplatesService
	// Streams manages message streams.
	Streams *StreamsService
	// Stats reads message and delivery-queue counters.
	Stats *StatsService
	// Bounces reads bounced messages.
	Bounces *BouncesService
	// DMARC reads stored DMARC aggregate reports and summaries.
	DMARC *DMARCService
	// Campaigns plans and sends broadcast campaigns.
	Campaigns *CampaignsService
	// Subscribers manages the opt-in audience of a broadcast stream.
	Subscribers *SubscribersService
	// Layouts manages the wrappers shared by templates.
	Layouts *LayoutsService
	// Inbound reads inbound and held messages.
	Inbound *InboundService
	// Logs reads the server's request log and tag index.
	Logs *LogsService
}

// Option configures a Client created by NewClient.
type Option func(*Client)

// WithBaseURL points the client at a self-hosted CamelMailer instance,
// e.g. "https://mail.example.com". A trailing slash is ignored.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// WithHTTPClient replaces the underlying *http.Client, e.g. to set
// timeouts, proxies, or custom transports.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// WithUserAgent overrides the User-Agent header sent with every request.
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// NewClient returns a Client authenticated with the given server API
// key (the X-Server-API-Key credential of one mail server).
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		userAgent:  defaultUserAgent,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.Emails = &EmailsService{client: c}
	c.Templates = &TemplatesService{client: c}
	c.Streams = &StreamsService{client: c}
	c.Stats = &StatsService{client: c}
	c.Bounces = &BouncesService{client: c}
	c.DMARC = &DMARCService{client: c}
	c.Campaigns = &CampaignsService{client: c}
	c.Subscribers = &SubscribersService{client: c}
	c.Layouts = &LayoutsService{client: c}
	c.Inbound = &InboundService{client: c}
	c.Logs = &LogsService{client: c}
	return c
}

// PingResult is the response of Client.Ping.
type PingResult struct {
	// Pong is true when the API key resolved to a server.
	Pong bool `json:"pong"`
	// ServerID is the numeric id of the authenticated server.
	ServerID int64 `json:"server_id"`
	// Server is the permalink of the authenticated server.
	Server string `json:"server"`
}

// Ping validates the API key against GET /api/v2/server/ping and
// reports which server it is scoped to.
func (c *Client) Ping(ctx context.Context) (*PingResult, error) {
	out := new(PingResult)
	if err := c.do(ctx, http.MethodGet, "/api/v2/server/ping", nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// envelope is the uniform CamelMailer response wrapper:
// {"status": "success"|"error", "time": …, "data": … | "error": …}.
type envelope struct {
	Status string          `json:"status"`
	Time   float64         `json:"time"`
	Data   json.RawMessage `json:"data"`
	Error  *APIError       `json:"error"`
}

// do performs one API request. body (if non-nil) is JSON-encoded; on a
// success envelope, data is decoded into out (if non-nil). Error
// envelopes and non-2xx responses become *APIError.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	return c.doWithHeader(ctx, method, path, query, nil, body, out)
}

// doWithHeader is do with extra request headers. Separate because only
// the send endpoints need one (Idempotency-Key), and that key belongs
// outside the body: the body is what the server hashes to recognise the
// same request.
func (c *Client) doWithHeader(ctx context.Context, method, path string, query url.Values, header http.Header, body, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("camelmailer: encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("camelmailer: building request: %w", err)
	}
	req.Header.Set("X-Server-API-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for name, values := range header {
		for _, value := range values {
			req.Header.Set(name, value)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("camelmailer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("camelmailer: reading response: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode >= 400 {
			return &APIError{
				Code:       "HTTPError",
				Message:    http.StatusText(resp.StatusCode),
				StatusCode: resp.StatusCode,
			}
		}
		return fmt.Errorf("camelmailer: decoding response: %w", err)
	}

	if env.Status == "error" || resp.StatusCode >= 400 {
		apiErr := env.Error
		if apiErr == nil {
			apiErr = &APIError{Code: "UnknownError", Message: "the API returned an error without details"}
		}
		apiErr.StatusCode = resp.StatusCode
		return apiErr
	}

	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("camelmailer: decoding response data: %w", err)
		}
	}
	return nil
}
