package camelmailer

import (
	"context"
	"net/http"
	"net/url"
)

// LayoutsService manages the wrappers shared by templates via
// /api/v2/server/layouts.
//
// A layout wraps every template that uses it, so header, footer and
// styling live in one place instead of in each template.
type LayoutsService struct {
	client *Client
}

// Layout is one template wrapper.
type Layout struct {
	// ID is the numeric layout id.
	ID int64 `json:"id"`
	// UUID is the stable unique identifier.
	UUID string `json:"uuid"`
	// Name is the display name.
	Name string `json:"name"`
	// Permalink identifies the layout in API calls.
	Permalink string `json:"permalink"`
	// HTMLWrapper wraps the HTML body; it embeds the body with
	// {{{ content }}}. Required when creating a layout.
	HTMLWrapper string `json:"html_wrapper"`
	// TextWrapper wraps the plain-text body. Empty when the layout has
	// none; the API answers with an explicit null there.
	TextWrapper string `json:"text_wrapper"`
}

// CreateLayoutRequest is the payload for LayoutsService.Create.
type CreateLayoutRequest struct {
	// Name is the display name (required).
	Name string `json:"name"`
	// Permalink overrides the generated permalink.
	Permalink string `json:"permalink,omitempty"`
	// HTMLWrapper has to embed the body with {{{ content }}}; anything
	// else is refused with ValidationError.
	HTMLWrapper string `json:"html_wrapper,omitempty"`
	// TextWrapper wraps the plain-text body.
	TextWrapper string `json:"text_wrapper,omitempty"`
}

// UpdateLayoutRequest changes the given fields of a layout. Nil fields
// are left unchanged; use String for the pointer fields.
type UpdateLayoutRequest struct {
	// Name replaces the display name.
	Name *string `json:"name,omitempty"`
	// HTMLWrapper replaces the HTML wrapper.
	HTMLWrapper *string `json:"html_wrapper,omitempty"`
	// TextWrapper replaces the plain-text wrapper.
	TextWrapper *string `json:"text_wrapper,omitempty"`
}

// layoutData is the {"layout": …} response wrapper.
type layoutData struct {
	Layout Layout `json:"layout"`
}

// layoutPath builds the path for one layout.
func layoutPath(permalink string) string {
	return "/api/v2/server/layouts/" + url.PathEscape(permalink)
}

// List returns all layouts of the server.
//
// API: GET /api/v2/server/layouts
func (s *LayoutsService) List(ctx context.Context) ([]Layout, error) {
	var out struct {
		Layouts []Layout `json:"layouts"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/layouts", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Layouts, nil
}

// Create adds a layout.
//
// API: POST /api/v2/server/layouts
func (s *LayoutsService) Create(ctx context.Context, req *CreateLayoutRequest) (*Layout, error) {
	out := new(layoutData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/layouts", nil, req, out); err != nil {
		return nil, err
	}
	return &out.Layout, nil
}

// Get returns one layout by permalink.
//
// API: GET /api/v2/server/layouts/{permalink}
func (s *LayoutsService) Get(ctx context.Context, permalink string) (*Layout, error) {
	out := new(layoutData)
	if err := s.client.do(ctx, http.MethodGet, layoutPath(permalink), nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Layout, nil
}

// Update changes the given fields of a layout.
//
// API: PATCH /api/v2/server/layouts/{permalink}
func (s *LayoutsService) Update(ctx context.Context, permalink string, req *UpdateLayoutRequest) (*Layout, error) {
	out := new(layoutData)
	if err := s.client.do(ctx, http.MethodPatch, layoutPath(permalink), nil, req, out); err != nil {
		return nil, err
	}
	return &out.Layout, nil
}

// Delete removes a layout. Templates that referenced it fall back to no
// wrapper.
//
// API: DELETE /api/v2/server/layouts/{permalink}
func (s *LayoutsService) Delete(ctx context.Context, permalink string) error {
	return s.client.do(ctx, http.MethodDelete, layoutPath(permalink), nil, nil, nil)
}

// UploadLogo stores the layout's logo, given as a data URL
// ("data:image/png;base64,…"), and returns the absolute URL to
// reference from the wrapper.
//
// API: POST /api/v2/server/layouts/{permalink}/logo
func (s *LayoutsService) UploadLogo(ctx context.Context, permalink, dataURL string) (string, error) {
	body := struct {
		DataURL string `json:"data_url"`
	}{DataURL: dataURL}
	var out struct {
		// The API names this key "url", not "logo_url".
		URL string `json:"url"`
	}
	if err := s.client.do(ctx, http.MethodPost, layoutPath(permalink)+"/logo", nil, body, &out); err != nil {
		return "", err
	}
	return out.URL, nil
}
