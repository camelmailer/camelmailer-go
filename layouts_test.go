package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

const layoutJSON = `{"id":1,"uuid":"l-1","name":"Default","permalink":"default",` +
	`"html_wrapper":"<html><body>{{{ content }}}</body></html>","text_wrapper":null}`

func TestLayoutsList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/layouts" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"layouts":[`+layoutJSON+`]}`)
	})
	layouts, err := client.Layouts.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(layouts) != 1 || layouts[0].Permalink != "default" {
		t.Errorf("layouts = %+v", layouts)
	}
}

func TestLayoutsCreate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"layout":`+layoutJSON+`}`)
	})
	layout, err := client.Layouts.Create(context.Background(), &CreateLayoutRequest{
		Name:        "Default",
		Permalink:   "default",
		HTMLWrapper: "<html><body>{{{ content }}}</body></html>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["html_wrapper"] != "<html><body>{{{ content }}}</body></html>" {
		t.Errorf("body = %v", gotBody)
	}
	if layout.Name != "Default" {
		t.Errorf("layout = %+v", layout)
	}
}

func TestLayoutsCreateWithoutContentPlaceholderIsRefused(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnprocessableEntity, "ValidationError", "html_wrapper must contain {{{ content }}}")
	})
	_, err := client.Layouts.Create(context.Background(), &CreateLayoutRequest{
		Name:        "Broken",
		HTMLWrapper: "<html></html>",
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "ValidationError" {
		t.Errorf("err = %v", err)
	}
}

func TestLayoutsGetUpdateDelete(t *testing.T) {
	var requests []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		success(t, w, http.StatusOK, `{"layout":`+layoutJSON+`}`)
	})
	ctx := context.Background()
	if _, err := client.Layouts.Get(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Layouts.Update(ctx, "default", &UpdateLayoutRequest{Name: String("Main")}); err != nil {
		t.Fatal(err)
	}
	if err := client.Layouts.Delete(ctx, "default"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /api/v2/server/layouts/default",
		"PATCH /api/v2/server/layouts/default",
		"DELETE /api/v2/server/layouts/default",
	}
	for i, w := range want {
		if requests[i] != w {
			t.Errorf("request %d = %s, want %s", i, requests[i], w)
		}
	}
}

func TestLayoutsUploadLogo(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/layouts/default/logo" {
			t.Errorf("path = %s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"url":"https://app.camelmailer.com/assets/layouts/l-1/logo"}`)
	})
	url, err := client.Layouts.UploadLogo(context.Background(), "default", "data:image/png;base64,iVBORw0KGgo=")
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["data_url"] != "data:image/png;base64,iVBORw0KGgo=" {
		t.Errorf("body = %v", gotBody)
	}
	// The API names this key "url"; reading "logo_url" would silently
	// return the empty string.
	if url != "https://app.camelmailer.com/assets/layouts/l-1/logo" {
		t.Errorf("url = %s", url)
	}
}
