package camelmailer

import (
	"context"
	"net/http"
	"net/url"
)

// TemplatesService manages stored message templates via
// /api/v2/server/templates. Template subjects and bodies may contain
// Mustache-style {{ variables }}.
type TemplatesService struct {
	client *Client
}

// Template is a stored message template.
type Template struct {
	// ID is the numeric template id.
	ID int64 `json:"id"`
	// UUID is the stable unique identifier.
	UUID string `json:"uuid"`
	// Name is the display name.
	Name string `json:"name"`
	// Permalink identifies the template in API calls.
	Permalink string `json:"permalink"`
	// Subject is the subject template.
	Subject string `json:"subject"`
	// HTMLBody is the HTML body template.
	HTMLBody string `json:"html_body"`
	// TextBody is the plain-text body template.
	TextBody string `json:"text_body"`
	// Archived reports whether the template is archived.
	Archived bool `json:"archived"`
}

// CreateTemplateRequest is the payload for TemplatesService.Create.
type CreateTemplateRequest struct {
	// Name is the display name (required).
	Name string `json:"name"`
	// Permalink overrides the generated permalink.
	Permalink string `json:"permalink,omitempty"`
	// Subject is the subject template.
	Subject string `json:"subject,omitempty"`
	// HTMLBody is the HTML body template.
	HTMLBody string `json:"html_body,omitempty"`
	// TextBody is the plain-text body template.
	TextBody string `json:"text_body,omitempty"`
}

// UpdateTemplateRequest is the payload for TemplatesService.Update.
// Nil fields are left unchanged; use String and Bool for the pointer
// fields.
type UpdateTemplateRequest struct {
	// Name replaces the display name.
	Name *string `json:"name,omitempty"`
	// Subject replaces the subject template.
	Subject *string `json:"subject,omitempty"`
	// HTMLBody replaces the HTML body template.
	HTMLBody *string `json:"html_body,omitempty"`
	// TextBody replaces the plain-text body template.
	TextBody *string `json:"text_body,omitempty"`
	// Archived archives or unarchives the template.
	Archived *bool `json:"archived,omitempty"`
}

// RenderedTemplate is a template rendered against a model. Fields are
// nil when the template has no corresponding part.
type RenderedTemplate struct {
	// Subject is the rendered subject.
	Subject *string `json:"subject"`
	// HTMLBody is the rendered HTML body.
	HTMLBody *string `json:"html_body"`
	// TextBody is the rendered plain-text body.
	TextBody *string `json:"text_body"`
}

// templateData is the {"template": …} response wrapper.
type templateData struct {
	Template Template `json:"template"`
}

// List returns all templates of the server.
//
// API: GET /api/v2/server/templates
func (s *TemplatesService) List(ctx context.Context) ([]Template, error) {
	var out struct {
		Templates []Template `json:"templates"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/templates", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Templates, nil
}

// Create stores a new template.
//
// API: POST /api/v2/server/templates
func (s *TemplatesService) Create(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	out := new(templateData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/templates", nil, req, out); err != nil {
		return nil, err
	}
	return &out.Template, nil
}

// Get returns one template by permalink.
//
// API: GET /api/v2/server/templates/{permalink}
func (s *TemplatesService) Get(ctx context.Context, permalink string) (*Template, error) {
	out := new(templateData)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/templates/"+url.PathEscape(permalink), nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Template, nil
}

// Update changes the given fields of a template.
//
// API: PATCH /api/v2/server/templates/{permalink}
func (s *TemplatesService) Update(ctx context.Context, permalink string, req *UpdateTemplateRequest) (*Template, error) {
	out := new(templateData)
	if err := s.client.do(ctx, http.MethodPatch, "/api/v2/server/templates/"+url.PathEscape(permalink), nil, req, out); err != nil {
		return nil, err
	}
	return &out.Template, nil
}

// Archive archives a template.
//
// API: POST /api/v2/server/templates/{permalink}/archive
func (s *TemplatesService) Archive(ctx context.Context, permalink string) (*Template, error) {
	out := new(templateData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/templates/"+url.PathEscape(permalink)+"/archive", nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Template, nil
}

// Render previews a template against model without sending anything.
// model may be nil.
//
// API: POST /api/v2/server/templates/{permalink}/render
func (s *TemplatesService) Render(ctx context.Context, permalink string, model map[string]any) (*RenderedTemplate, error) {
	body := struct {
		TemplateModel map[string]any `json:"template_model,omitempty"`
	}{TemplateModel: model}
	var out struct {
		Rendered RenderedTemplate `json:"rendered"`
	}
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/templates/"+url.PathEscape(permalink)+"/render", nil, body, &out); err != nil {
		return nil, err
	}
	return &out.Rendered, nil
}
