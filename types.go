package camelmailer

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

// Address is an email address with an optional display name. It
// marshals to a bare string ("ada@example.com") when Name is empty and
// to {"email": …, "name": …} otherwise, and unmarshals from either
// form.
type Address struct {
	// Email is the address itself (required).
	Email string `json:"email"`
	// Name is the optional display name.
	Name string `json:"name,omitempty"`
}

// MarshalJSON implements json.Marshaler.
func (a Address) MarshalJSON() ([]byte, error) {
	if a.Name == "" {
		return json.Marshal(a.Email)
	}
	type plain Address
	return json.Marshal(plain(a))
}

// UnmarshalJSON implements json.Unmarshaler, accepting both a bare
// email string and an {email, name} object.
func (a *Address) UnmarshalJSON(data []byte) error {
	var email string
	if err := json.Unmarshal(data, &email); err == nil {
		a.Email = email
		a.Name = ""
		return nil
	}
	type plain Address
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*a = Address(p)
	return nil
}

// Attachment is a file attached to an outgoing message. Data is
// base64-encoded on the wire automatically.
type Attachment struct {
	// Name is the file name, e.g. "invoice.pdf".
	Name string `json:"name"`
	// ContentType is the MIME type, e.g. "application/pdf".
	ContentType string `json:"content_type"`
	// Data is the raw file content.
	Data []byte `json:"data_base64"`
}

// Pagination describes the page window of a list response.
type Pagination struct {
	// Page is the 1-based page number.
	Page int `json:"page"`
	// PerPage is the page size (1–100, default 25).
	PerPage int `json:"per_page"`
	// Total is the total number of items.
	Total int64 `json:"total"`
	// TotalPages is the total number of pages.
	TotalPages int64 `json:"total_pages"`
}

// ListOptions selects a page of a paginated list endpoint.
type ListOptions struct {
	// Page is the 1-based page number (default 1).
	Page int
	// PerPage is the page size, capped at 100 (default 25).
	PerPage int
}

// String returns a pointer to s, for optional request fields.
func String(s string) *string { return &s }

// Bool returns a pointer to b, for optional request fields.
func Bool(b bool) *bool { return &b }

// values appends the pagination parameters to q.
func (o ListOptions) values(q url.Values) {
	if o.Page > 0 {
		q.Set("page", strconv.Itoa(o.Page))
	}
	if o.PerPage > 0 {
		q.Set("per_page", strconv.Itoa(o.PerPage))
	}
}

// setTime adds a RFC 3339 time parameter to q when t is non-zero.
func setTime(q url.Values, key string, t time.Time) {
	if !t.IsZero() {
		q.Set(key, t.UTC().Format(time.RFC3339))
	}
}
