package camelmailer

import (
	"context"
	"net/http"
	"net/url"
)

// StreamsService manages message streams via /api/v2/server/streams.
type StreamsService struct {
	client *Client
}

// Stream is a message stream of the server.
type Stream struct {
	// ID is the numeric stream id.
	ID int64 `json:"id"`
	// UUID is the stable unique identifier.
	UUID string `json:"uuid"`
	// Name is the display name.
	Name string `json:"name"`
	// Permalink identifies the stream in API calls and send requests.
	Permalink string `json:"permalink"`
	// StreamType is "transactional", "broadcast" or "inbound".
	StreamType string `json:"stream_type"`
	// Archived reports whether the stream is archived.
	Archived bool `json:"archived"`
}

// CreateStreamRequest is the payload for StreamsService.Create.
type CreateStreamRequest struct {
	// Name is the display name (required).
	Name string `json:"name"`
	// Permalink overrides the generated permalink.
	Permalink string `json:"permalink,omitempty"`
	// StreamType is "transactional" (default), "broadcast" or
	// "inbound".
	StreamType string `json:"stream_type,omitempty"`
}

// UpdateStreamRequest is the payload for StreamsService.Update. Nil
// fields are left unchanged; use String and Bool for the pointer
// fields.
type UpdateStreamRequest struct {
	// Name replaces the display name.
	Name *string `json:"name,omitempty"`
	// StreamType replaces the stream type.
	StreamType *string `json:"stream_type,omitempty"`
	// Archived archives or unarchives the stream.
	Archived *bool `json:"archived,omitempty"`
}

// streamData is the {"stream": …} response wrapper.
type streamData struct {
	Stream Stream `json:"stream"`
}

// List returns all message streams of the server.
//
// API: GET /api/v2/server/streams
func (s *StreamsService) List(ctx context.Context) ([]Stream, error) {
	var out struct {
		Streams []Stream `json:"streams"`
	}
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/streams", nil, nil, &out); err != nil {
		return nil, err
	}
	return out.Streams, nil
}

// Create adds a message stream.
//
// API: POST /api/v2/server/streams
func (s *StreamsService) Create(ctx context.Context, req *CreateStreamRequest) (*Stream, error) {
	out := new(streamData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/streams", nil, req, out); err != nil {
		return nil, err
	}
	return &out.Stream, nil
}

// Get returns one stream by permalink.
//
// API: GET /api/v2/server/streams/{permalink}
func (s *StreamsService) Get(ctx context.Context, permalink string) (*Stream, error) {
	out := new(streamData)
	if err := s.client.do(ctx, http.MethodGet, "/api/v2/server/streams/"+url.PathEscape(permalink), nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Stream, nil
}

// Update changes the given fields of a stream.
//
// API: PATCH /api/v2/server/streams/{permalink}
func (s *StreamsService) Update(ctx context.Context, permalink string, req *UpdateStreamRequest) (*Stream, error) {
	out := new(streamData)
	if err := s.client.do(ctx, http.MethodPatch, "/api/v2/server/streams/"+url.PathEscape(permalink), nil, req, out); err != nil {
		return nil, err
	}
	return &out.Stream, nil
}

// Archive archives a stream; archived streams reject new messages.
//
// API: POST /api/v2/server/streams/{permalink}/archive
func (s *StreamsService) Archive(ctx context.Context, permalink string) (*Stream, error) {
	out := new(streamData)
	if err := s.client.do(ctx, http.MethodPost, "/api/v2/server/streams/"+url.PathEscape(permalink)+"/archive", nil, nil, out); err != nil {
		return nil, err
	}
	return &out.Stream, nil
}
