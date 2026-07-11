package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

const streamJSON = `{"id":1,"uuid":"s-1","name":"Default","permalink":"default","stream_type":"transactional","archived":false}`

func TestStreamsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/server/streams" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"streams":[`+streamJSON+`]}`)
	})
	streams, err := client.Streams.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(streams) != 1 || streams[0].Permalink != "default" || streams[0].StreamType != "transactional" {
		t.Errorf("streams = %+v", streams)
	}
}

func TestStreamsCreate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/streams" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"stream":{"id":2,"uuid":"s-2","name":"Broadcasts","permalink":"broadcasts","stream_type":"broadcast","archived":false}}`)
	})
	stream, err := client.Streams.Create(context.Background(), &CreateStreamRequest{
		Name:       "Broadcasts",
		StreamType: "broadcast",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "Broadcasts" || gotBody["stream_type"] != "broadcast" {
		t.Errorf("body = %v", gotBody)
	}
	if stream.ID != 2 || stream.StreamType != "broadcast" {
		t.Errorf("stream = %+v", stream)
	}
}

func TestStreamsCreateConflict(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnprocessableEntity, "ValidationError", `permalink "broadcasts" is already taken`)
	})
	_, err := client.Streams.Create(context.Background(), &CreateStreamRequest{Name: "Broadcasts"})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "ValidationError" {
		t.Fatalf("err = %v", err)
	}
}

func TestStreamsGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/streams/default" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"stream":`+streamJSON+`}`)
	})
	stream, err := client.Streams.Get(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Name != "Default" {
		t.Errorf("stream = %+v", stream)
	}
}

func TestStreamsUpdate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/server/streams/default" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"stream":`+streamJSON+`}`)
	})
	_, err := client.Streams.Update(context.Background(), "default", &UpdateStreamRequest{
		Name: String("Renamed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "Renamed" {
		t.Errorf("body = %v", gotBody)
	}
	if _, present := gotBody["stream_type"]; present {
		t.Error("unset fields must be omitted from the PATCH body")
	}
}

func TestStreamsArchive(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/streams/default/archive" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"stream":{"id":1,"uuid":"s-1","name":"Default","permalink":"default","stream_type":"transactional","archived":true}}`)
	})
	stream, err := client.Streams.Archive(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Archived {
		t.Errorf("stream = %+v", stream)
	}
}

func TestStreamsGetNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotFound", "Resource not found")
	})
	_, err := client.Streams.Get(context.Background(), "missing")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotFound" || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v", err)
	}
}
