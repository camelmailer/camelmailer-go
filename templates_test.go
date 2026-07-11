package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

const templateJSON = `{
	"id": 3, "uuid": "u-3", "name": "Welcome", "permalink": "welcome",
	"subject": "Hello {{ name }}", "html_body": "<p>Hi {{ name }}</p>",
	"text_body": "Hi {{ name }}", "archived": false
}`

func TestTemplatesList(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/server/templates" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"templates":[`+templateJSON+`]}`)
	})
	templates, err := client.Templates.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) != 1 || templates[0].Permalink != "welcome" {
		t.Errorf("templates = %+v", templates)
	}
}

func TestTemplatesCreate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/templates" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusCreated, `{"template":`+templateJSON+`}`)
	})
	template, err := client.Templates.Create(context.Background(), &CreateTemplateRequest{
		Name:     "Welcome",
		Subject:  "Hello {{ name }}",
		HTMLBody: "<p>Hi {{ name }}</p>",
		TextBody: "Hi {{ name }}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["name"] != "Welcome" || gotBody["subject"] != "Hello {{ name }}" {
		t.Errorf("body = %v", gotBody)
	}
	if _, present := gotBody["permalink"]; present {
		t.Error("empty permalink must be omitted")
	}
	if template.ID != 3 || template.Name != "Welcome" {
		t.Errorf("template = %+v", template)
	}
}

func TestTemplatesCreateMissingName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusBadRequest, "ParameterMissing", "param is missing or the value is empty: name")
	})
	_, err := client.Templates.Create(context.Background(), &CreateTemplateRequest{})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "ParameterMissing" || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("err = %v", err)
	}
}

func TestTemplatesGet(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/templates/welcome" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"template":`+templateJSON+`}`)
	})
	template, err := client.Templates.Get(context.Background(), "welcome")
	if err != nil {
		t.Fatal(err)
	}
	if template.UUID != "u-3" {
		t.Errorf("template = %+v", template)
	}
}

func TestTemplatesUpdate(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/server/templates/welcome" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"template":`+templateJSON+`}`)
	})
	_, err := client.Templates.Update(context.Background(), "welcome", &UpdateTemplateRequest{
		Subject: String("New subject"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody["subject"] != "New subject" {
		t.Errorf("body = %v", gotBody)
	}
	if _, present := gotBody["name"]; present {
		t.Error("unset fields must be omitted from the PATCH body")
	}
}

func TestTemplatesArchive(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/templates/welcome/archive" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"template":{"id":3,"uuid":"u-3","name":"Welcome","permalink":"welcome","archived":true}}`)
	})
	template, err := client.Templates.Archive(context.Background(), "welcome")
	if err != nil {
		t.Fatal(err)
	}
	if !template.Archived {
		t.Errorf("template = %+v", template)
	}
}

func TestTemplatesRender(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/server/templates/welcome/render" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		success(t, w, http.StatusOK, `{"rendered":{"subject":"Hello Ada","html_body":"<p>Hi Ada</p>","text_body":null}}`)
	})
	rendered, err := client.Templates.Render(context.Background(), "welcome", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	model := gotBody["template_model"].(map[string]any)
	if model["name"] != "Ada" {
		t.Errorf("body = %v", gotBody)
	}
	if rendered.Subject == nil || *rendered.Subject != "Hello Ada" {
		t.Errorf("rendered = %+v", rendered)
	}
	if rendered.TextBody != nil {
		t.Errorf("TextBody = %v, want nil for JSON null", *rendered.TextBody)
	}
}

func TestTemplatesRenderNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusNotFound, "NotFound", "Resource not found")
	})
	_, err := client.Templates.Render(context.Background(), "missing", nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NotFound" {
		t.Fatalf("err = %v", err)
	}
}

func TestTemplatePermalinkIsEscaped(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/server/templates/wel come" {
			t.Errorf("path = %s", r.URL.Path)
		}
		success(t, w, http.StatusOK, `{"template":`+templateJSON+`}`)
	})
	if _, err := client.Templates.Get(context.Background(), "wel come"); err != nil {
		t.Fatal(err)
	}
}
