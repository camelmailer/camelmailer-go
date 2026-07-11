package camelmailer

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// EmailsService sends and reads messages via /api/v2/server/messages.
type EmailsService struct {
	client *Client
}

// SendEmailRequest is the payload for EmailsService.Send. From and at
// least one recipient (To, Cc or Bcc) are required; the From domain
// must be a verified sending domain (or confirmed sender address) of
// the server.
type SendEmailRequest struct {
	// From is the sender address.
	From Address `json:"from"`
	// To lists the primary recipients.
	To []Address `json:"to,omitempty"`
	// Cc lists the carbon-copy recipients.
	Cc []Address `json:"cc,omitempty"`
	// Bcc lists the blind-carbon-copy recipients.
	Bcc []Address `json:"bcc,omitempty"`
	// ReplyTo lists the Reply-To addresses.
	ReplyTo []Address `json:"reply_to,omitempty"`
	// Subject is the message subject.
	Subject string `json:"subject,omitempty"`
	// HTMLBody is the HTML part.
	HTMLBody string `json:"html_body,omitempty"`
	// TextBody is the plain-text part.
	TextBody string `json:"text_body,omitempty"`
	// Headers sets extra message headers.
	Headers map[string]string `json:"headers,omitempty"`
	// Attachments lists file attachments.
	Attachments []Attachment `json:"attachments,omitempty"`
	// Tag is a free-form tag for filtering and stats.
	Tag string `json:"tag,omitempty"`
	// Metadata is arbitrary JSON stored with the message.
	Metadata map[string]any `json:"metadata,omitempty"`
	// Stream is a message-stream permalink; defaults to the server's
	// default stream.
	Stream string `json:"stream,omitempty"`
}

// SendWithTemplateRequest is the payload for
// EmailsService.SendWithTemplate. The embedded SendEmailRequest fields
// are flattened on the wire; fields set directly (e.g. Subject)
// override the rendered template fields.
type SendWithTemplateRequest struct {
	SendEmailRequest
	// Template is the permalink of the stored template (required).
	Template string `json:"template"`
	// TemplateModel provides the values for the template's
	// {{ variables }}.
	TemplateModel map[string]any `json:"template_model,omitempty"`
}

// SendRecipient is the per-recipient outcome of a send.
type SendRecipient struct {
	// RcptTo is the recipient address.
	RcptTo string `json:"rcpt_to"`
	// MessageID is the id of the stored message for this recipient.
	MessageID int64 `json:"message_id"`
	// Token is the public token of the stored message.
	Token string `json:"token"`
	// Status is the queue status, e.g. "queued".
	Status string `json:"status"`
}

// SendResult is the response of a single send.
type SendResult struct {
	// MessageID is the id of the first stored message.
	MessageID int64 `json:"message_id"`
	// Recipients holds one entry per recipient.
	Recipients []SendRecipient `json:"recipients"`
}

// BatchEntryResult is the outcome of one entry in a batch send. Either
// Data (Status "success") or Error (Status "error") is set.
type BatchEntryResult struct {
	// Status is "success" or "error".
	Status string `json:"status"`
	// Data is the send result when Status is "success".
	Data *SendResult `json:"data,omitempty"`
	// Error describes the failure when Status is "error".
	Error *APIError `json:"error,omitempty"`
}

// BatchSendResult is the response of a batch send; entries are in
// request order.
type BatchSendResult struct {
	// Messages holds one result per submitted message.
	Messages []BatchEntryResult `json:"messages"`
}

// Message is a stored (incoming or outgoing) message.
type Message struct {
	// ID is the numeric message id.
	ID int64 `json:"id"`
	// Token is the public message token.
	Token string `json:"token"`
	// Scope is "incoming" or "outgoing".
	Scope string `json:"scope"`
	// RcptTo is the recipient address.
	RcptTo string `json:"rcpt_to"`
	// MailFrom is the envelope sender.
	MailFrom string `json:"mail_from"`
	// Subject is the message subject.
	Subject string `json:"subject"`
	// MessageID is the Message-ID header value.
	MessageID string `json:"message_id"`
	// Tag is the message tag.
	Tag string `json:"tag"`
	// Status is the delivery status, e.g. "Sent".
	Status string `json:"status"`
	// Bounce reports whether this message is a bounce.
	Bounce bool `json:"bounce"`
	// SpamStatus is the spam-check verdict.
	SpamStatus string `json:"spam_status"`
	// SpamScore is the spam-check score.
	SpamScore float64 `json:"spam_score"`
	// Held reports whether the message is held.
	Held bool `json:"held"`
	// Threat reports whether a threat was detected.
	Threat bool `json:"threat"`
	// Size is the message size in bytes.
	Size int64 `json:"size"`
	// Metadata is the arbitrary JSON stored with the message.
	Metadata map[string]any `json:"metadata"`
	// StreamID is the id of the message stream.
	StreamID int64 `json:"stream_id"`
	// Bypassed reports whether hold rules were bypassed.
	Bypassed bool `json:"bypassed"`
	// CreatedAt is the creation timestamp.
	CreatedAt time.Time `json:"created_at"`
}

// Delivery is one delivery attempt of a message.
type Delivery struct {
	// ID is the numeric delivery id.
	ID int64 `json:"id"`
	// Status is the attempt outcome, e.g. "Sent" or "SoftFail".
	Status string `json:"status"`
	// Details is the human-readable outcome description.
	Details string `json:"details"`
	// Output is the remote server output.
	Output string `json:"output"`
	// SentWithSSL reports whether TLS was used.
	SentWithSSL bool `json:"sent_with_ssl"`
	// CreatedAt is the attempt timestamp.
	CreatedAt time.Time `json:"created_at"`
}

// ActivityEvent is one open or click event of a message.
type ActivityEvent struct {
	// IPAddress is the client IP.
	IPAddress string `json:"ip_address"`
	// UserAgent is the client User-Agent.
	UserAgent string `json:"user_agent"`
	// URL is the clicked link (empty for opens).
	URL string `json:"url"`
	// CreatedAt is the event timestamp.
	CreatedAt time.Time `json:"created_at"`
}

// Email is one message with its delivery attempts, as returned by
// EmailsService.Get.
type Email struct {
	// Message is the stored message.
	Message Message `json:"message"`
	// Deliveries lists its delivery attempts.
	Deliveries []Delivery `json:"deliveries"`
}

// ListEmailsOptions filters EmailsService.List.
type ListEmailsOptions struct {
	ListOptions
	// Scope restricts to "incoming" or "outgoing" messages.
	Scope string
	// Status restricts to one delivery status.
	Status string
	// Tag restricts to one tag.
	Tag string
	// Query is a substring match on subject and addresses.
	Query string
	// Stream restricts to one message stream (by permalink).
	Stream string
}

// ListEmailsResult is one page of messages.
type ListEmailsResult struct {
	// Messages is the page of messages.
	Messages []Message `json:"messages"`
	// Pagination describes the page window.
	Pagination Pagination `json:"pagination"`
}

// Send queues one message per recipient.
//
// API: POST /api/v2/server/messages
func (s *EmailsService) Send(ctx context.Context, req *SendEmailRequest) (*SendResult, error) {
	out := new(SendResult)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/messages", nil, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendBatch queues many messages in one call and returns one result
// per entry; individual entries can fail without failing the batch.
//
// API: POST /api/v2/server/messages/batch
func (s *EmailsService) SendBatch(ctx context.Context, reqs []*SendEmailRequest) (*BatchSendResult, error) {
	out := new(BatchSendResult)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/messages/batch", nil, reqs, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendWithTemplate renders a stored template against
// req.TemplateModel and sends the result.
//
// API: POST /api/v2/server/messages/with_template
func (s *EmailsService) SendWithTemplate(ctx context.Context, req *SendWithTemplateRequest) (*SendResult, error) {
	out := new(SendResult)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/messages/with_template", nil, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SendWithTemplateBatch sends a stored template to many recipients in
// one call and returns one result per entry.
//
// API: POST /api/v2/server/messages/with_template/batch
func (s *EmailsService) SendWithTemplateBatch(ctx context.Context, reqs []*SendWithTemplateRequest) (*BatchSendResult, error) {
	out := new(BatchSendResult)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/messages/with_template/batch", nil, reqs, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get returns one message with its delivery attempts.
//
// API: GET /api/v2/server/messages/{id}
func (s *EmailsService) Get(ctx context.Context, id int64) (*Email, error) {
	out := new(Email)
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/messages/%d", id), nil, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// List returns the server's messages, filtered and paginated. opts may
// be nil.
//
// API: GET /api/v2/server/messages
func (s *EmailsService) List(ctx context.Context, opts *ListEmailsOptions) (*ListEmailsResult, error) {
	query := url.Values{}
	if opts != nil {
		opts.ListOptions.values(query)
		for key, value := range map[string]string{
			"scope":  opts.Scope,
			"status": opts.Status,
			"tag":    opts.Tag,
			"query":  opts.Query,
			"stream": opts.Stream,
		} {
			if value != "" {
				query.Set(key, value)
			}
		}
	}
	out := new(ListEmailsResult)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/messages", query, nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Deliveries returns the delivery attempts of a message.
//
// API: GET /api/v2/server/messages/{id}/deliveries
func (s *EmailsService) Deliveries(ctx context.Context, id int64) ([]Delivery, error) {
	var out struct {
		Deliveries []Delivery `json:"deliveries"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/messages/%d/deliveries", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Deliveries, nil
}

// Opens returns the open events of a message.
//
// API: GET /api/v2/server/messages/{id}/opens
func (s *EmailsService) Opens(ctx context.Context, id int64) ([]ActivityEvent, error) {
	var out struct {
		Opens []ActivityEvent `json:"opens"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/messages/%d/opens", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Opens, nil
}

// Clicks returns the click events of a message.
//
// API: GET /api/v2/server/messages/{id}/clicks
func (s *EmailsService) Clicks(ctx context.Context, id int64) ([]ActivityEvent, error) {
	var out struct {
		Clicks []ActivityEvent `json:"clicks"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/messages/%d/clicks", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Clicks, nil
}

// Raw returns the raw RFC 5322 source of a message, decoded from the
// API's base64 representation. Servers in privacy mode return a
// NotAvailable error instead.
//
// API: GET /api/v2/server/messages/{id}/raw
func (s *EmailsService) Raw(ctx context.Context, id int64) ([]byte, error) {
	var out struct {
		RawMessage string `json:"raw_message"`
	}
	if err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/v2/server/messages/%d/raw", id), nil, nil, &out); err != nil {
		return nil, err
	}
	raw, err := base64.StdEncoding.DecodeString(out.RawMessage)
	if err != nil {
		return nil, fmt.Errorf("camelmailer: decoding raw message: %w", err)
	}
	return raw, nil
}
