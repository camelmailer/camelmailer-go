package camelmailer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient starts an httptest.Server around handler and returns a
// Client pointed at it. The server is closed via t.Cleanup.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient("cm_test_key", WithBaseURL(server.URL))
}

// success writes a success envelope with the given data payload.
func success(t *testing.T, w http.ResponseWriter, status int, data string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"status":"success","time":0.004,"data":` + data + `}`))
}

// failure writes an error envelope with the given code and message.
func failure(t *testing.T, w http.ResponseWriter, status int, code, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body, _ := json.Marshal(map[string]any{
		"status": "error",
		"time":   0.004,
		"error":  map[string]string{"code": code, "message": message},
	})
	_, _ = w.Write(body)
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient("cm_key")
	if client.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", client.baseURL, DefaultBaseURL)
	}
	if client.httpClient == nil {
		t.Fatal("httpClient is nil")
	}
	for name, svc := range map[string]any{
		"Emails":    client.Emails,
		"Templates": client.Templates,
		"Streams":   client.Streams,
		"Stats":     client.Stats,
		"Bounces":   client.Bounces,
		"DMARC":     client.DMARC,
	} {
		if svc == nil {
			t.Errorf("client.%s is nil", name)
		}
	}
}

func TestWithBaseURLTrimsTrailingSlash(t *testing.T) {
	client := NewClient("cm_key", WithBaseURL("https://mail.example.com/"))
	if client.baseURL != "https://mail.example.com" {
		t.Errorf("baseURL = %q, want trailing slash trimmed", client.baseURL)
	}
}

func TestWithHTTPClient(t *testing.T) {
	custom := &http.Client{Timeout: 5 * time.Second}
	client := NewClient("cm_key", WithHTTPClient(custom))
	if client.httpClient != custom {
		t.Error("WithHTTPClient did not set the http client")
	}
}

func TestWithUserAgent(t *testing.T) {
	var got string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		success(t, w, http.StatusOK, `{"pong":true,"server_id":1,"server":"acme"}`)
	})
	WithUserAgent("my-app/1.0")(client)
	if _, err := client.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "my-app/1.0" {
		t.Errorf("User-Agent = %q, want %q", got, "my-app/1.0")
	}
}

func TestRequestCarriesAuthHeaders(t *testing.T) {
	var (
		gotKey    string
		gotAccept string
		gotUA     string
	)
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Server-API-Key")
		gotAccept = r.Header.Get("Accept")
		gotUA = r.Header.Get("User-Agent")
		success(t, w, http.StatusOK, `{"pong":true,"server_id":42,"server":"acme"}`)
	})

	pong, err := client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "cm_test_key" {
		t.Errorf("X-Server-API-Key = %q, want %q", gotKey, "cm_test_key")
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q", gotAccept)
	}
	if gotUA != defaultUserAgent {
		t.Errorf("User-Agent = %q, want %q", gotUA, defaultUserAgent)
	}
	if !pong.Pong || pong.ServerID != 42 || pong.Server != "acme" {
		t.Errorf("unexpected ping result: %+v", pong)
	}
}

func TestAPIErrorIsReturnedTyped(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusUnauthorized, "Unauthorized", "Invalid server API token")
	})

	_, err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.Code != "Unauthorized" {
		t.Errorf("Code = %q", apiErr.Code)
	}
	if apiErr.Message != "Invalid server API token" {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
}

func TestAPIErrorErrorString(t *testing.T) {
	err := &APIError{Code: "ValidationError", Message: "boom", StatusCode: 422}
	want := "camelmailer: ValidationError: boom (HTTP 422)"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestEnvelopeErrorWithOKStatusCode(t *testing.T) {
	// A body-level error envelope must surface even if HTTP status is 200.
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		failure(t, w, http.StatusOK, "ValidationError", "bad input")
	})
	_, err := client.Ping(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.Code != "ValidationError" || apiErr.StatusCode != http.StatusOK {
		t.Errorf("unexpected error: %+v", apiErr)
	}
}

func TestNonJSONErrorResponse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	})
	_, err := client.Ping(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Errorf("StatusCode = %d", apiErr.StatusCode)
	}
}

func TestContextCancellation(t *testing.T) {
	block := make(chan struct{})
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-block
	})
	t.Cleanup(func() { close(block) })

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Ping(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestAddressMarshalJSON(t *testing.T) {
	bare, err := json.Marshal(Address{Email: "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if string(bare) != `"ada@example.com"` {
		t.Errorf("bare address = %s, want plain string", bare)
	}

	named, err := json.Marshal(Address{Email: "ada@example.com", Name: "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if string(named) != `{"email":"ada@example.com","name":"Ada"}` {
		t.Errorf("named address = %s", named)
	}
}

func TestAddressUnmarshalJSON(t *testing.T) {
	var fromString Address
	if err := json.Unmarshal([]byte(`"ada@example.com"`), &fromString); err != nil {
		t.Fatal(err)
	}
	if fromString.Email != "ada@example.com" || fromString.Name != "" {
		t.Errorf("fromString = %+v", fromString)
	}

	var fromObject Address
	if err := json.Unmarshal([]byte(`{"email":"ada@example.com","name":"Ada"}`), &fromObject); err != nil {
		t.Fatal(err)
	}
	if fromObject.Email != "ada@example.com" || fromObject.Name != "Ada" {
		t.Errorf("fromObject = %+v", fromObject)
	}
}

func TestAttachmentMarshalsBase64(t *testing.T) {
	data, err := json.Marshal(Attachment{
		Name:        "hello.txt",
		ContentType: "text/plain",
		Data:        []byte("hello"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"name":"hello.txt","content_type":"text/plain","data_base64":"aGVsbG8="}`
	if string(data) != want {
		t.Errorf("attachment = %s, want %s", data, want)
	}
}
