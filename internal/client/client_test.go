package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient returns a Client wired to the given test server.
// maxRetries is set to 0 to keep unit tests fast.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := New(Config{
		BaseURL:        srv.URL,
		Token:          "test-token",
		RequestTimeout: 5 * time.Second,
		MaxRetries:     0,
	})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	return c
}

// ---- New() validation ----

func TestNew_EmptyBaseURL(t *testing.T) {
	_, err := New(Config{Token: "tok"})
	if err == nil {
		t.Fatal("expected error for empty BaseURL, got nil")
	}
}

func TestNew_EmptyToken(t *testing.T) {
	_, err := New(Config{BaseURL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error for empty Token, got nil")
	}
}

func TestNew_Defaults(t *testing.T) {
	c, err := New(Config{BaseURL: "https://example.com", Token: "tok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.maxRetries != defaultMaxRetries {
		t.Errorf("maxRetries = %d, want %d", c.maxRetries, defaultMaxRetries)
	}
	if c.httpClient.Timeout != defaultTimeout {
		t.Errorf("timeout = %v, want %v", c.httpClient.Timeout, defaultTimeout)
	}
}

// ---- APIError ----

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 404, Body: "not found"}
	want := "MSL5 API error (HTTP 404): not found"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// ---- IsNotFound ----

func TestIsNotFound_True(t *testing.T) {
	err := &APIError{StatusCode: http.StatusNotFound}
	if !IsNotFound(err) {
		t.Error("IsNotFound should be true for 404")
	}
}

func TestIsNotFound_False_OtherStatus(t *testing.T) {
	err := &APIError{StatusCode: http.StatusBadRequest}
	if IsNotFound(err) {
		t.Error("IsNotFound should be false for 400")
	}
}

func TestIsNotFound_False_NonAPIError(t *testing.T) {
	err := errors.New("network error")
	if IsNotFound(err) {
		t.Error("IsNotFound should be false for non-APIError")
	}
}

// ---- Authorization header ----

func TestDo_SendsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer test-token")
	}
}

// ---- HTTP 4xx non-retryable ----

func TestDo_Returns4xxAsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for 400, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusBadRequest)
	}
}

// ---- Context cancellation ----

func TestDo_RespectsContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Simulate a slow handler; the client's context is already cancelled.
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err := c.do(ctx, http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

// ---- backoffDuration ----

func TestBackoffDuration(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{100, 30 * time.Second}, // capped at backoffMax
	}
	for _, tc := range cases {
		got := backoffDuration(tc.attempt)
		if got != tc.want {
			t.Errorf("backoffDuration(%d) = %v, want %v", tc.attempt, got, tc.want)
		}
	}
}

// newClientWithRetries returns a Client with an explicit maxRetries value.
// Pass maxRetries > 0 to bypass the "0 means use default" fallback in New().
func newClientWithRetries(t *testing.T, srv *httptest.Server, maxRetries int) *Client {
	t.Helper()
	c, err := New(Config{
		BaseURL:        srv.URL,
		Token:          "test-token",
		RequestTimeout: 5 * time.Second,
		MaxRetries:     maxRetries,
	})
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}
	return c
}

// ---- Retry: 5xx is retried ----
// Note: each retry incurs a 1 s backoff; these tests take ~1 s each.

func TestDo_5xx_IsRetried(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`error`))
	}))
	defer srv.Close()

	c := newClientWithRetries(t, srv, 1) // 1 retry → 2 total calls
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for persistent 500, got nil")
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2 (1 initial + 1 retry)", calls)
	}
}

// ---- Retry: 408 is retried (not treated as non-retryable 4xx) ----

func TestDo_408_IsRetried(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusRequestTimeout)
		_, _ = w.Write([]byte(`timeout`))
	}))
	defer srv.Close()

	c := newClientWithRetries(t, srv, 1)
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for persistent 408, got nil")
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
}

// ---- Retry: 429 is retried ----

func TestDo_429_IsRetried(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`rate limited`))
	}))
	defer srv.Close()

	c := newClientWithRetries(t, srv, 1)
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for persistent 429, got nil")
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
}

// ---- Retry: succeeds after a transient 5xx ----

func TestDo_RetrySucceedsAfterTransient5xx(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`error`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newClientWithRetries(t, srv, 1)
	body, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if calls != 2 {
		t.Errorf("server called %d times, want 2", calls)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", body, `{"ok":true}`)
	}
}

// ---- Retry: non-retryable 4xx is never retried ----

func TestDo_4xx_NonRetryable_NotRetried(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`forbidden`))
	}))
	defer srv.Close()

	c := newClientWithRetries(t, srv, 2) // generous retries, but 403 must not be retried
	_, _, err := c.do(context.Background(), http.MethodGet, "/test", nil)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if calls != 1 {
		t.Errorf("server called %d times, want 1 (no retry for 403)", calls)
	}
}
