package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

// ---- ListOrigins ----

func TestListOrigins_WrappedResponse(t *testing.T) {
	origins := []models.Origin{
		{ID: "o1", HostName: "host1.example.com", ContractID: "c1"},
		{ID: "o2", HostName: "host2.example.com", ContractID: "c1"},
	}
	payload, _ := json.Marshal(models.ListOriginsResponse{Origins: origins})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/origins" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListOrigins(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListOrigins() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(origins) = %d, want 2", len(got))
	}
	if got[0].ID != "o1" {
		t.Errorf("origins[0].ID = %q, want %q", got[0].ID, "o1")
	}
}

func TestListOrigins_DirectArrayResponse(t *testing.T) {
	origins := []models.Origin{{ID: "o1", HostName: "h.example.com"}}
	payload, _ := json.Marshal(origins)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListOrigins(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListOrigins() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "o1" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListOrigins_ForwardsQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"origins":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ListOrigins(context.Background(), "ctr1", "grp1")
	if err != nil {
		t.Fatalf("ListOrigins() unexpected error: %v", err)
	}
	if gotQuery == "" {
		t.Error("expected query params to be forwarded, got empty string")
	}
}

func TestListOrigins_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`unauthorized`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ListOrigins(context.Background(), "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- GetOrigin ----

func TestGetOrigin_Success(t *testing.T) {
	origin := models.Origin{ID: "abc123", HostName: "host.example.com", ContractID: "c1"}
	payload, _ := json.Marshal(origin)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/origins/abc123" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetOrigin(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("GetOrigin() unexpected error: %v", err)
	}
	if got.ID != "abc123" {
		t.Errorf("ID = %q, want %q", got.ID, "abc123")
	}
}

func TestGetOrigin_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetOrigin(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound() = false, want true; err = %v", err)
	}
}

// ---- CreateOrigin ----

func TestCreateOrigin_Success(t *testing.T) {
	req := models.OriginCreateRequest{
		HostName:       "host.example.com",
		IngestLocation: "US_ORD",
		ContractID:     "c1",
		CPTag:          "tag1",
		GroupID:        "g1",
	}
	resp := models.Origin{ID: "new-id", HostName: req.HostName}
	payload, _ := json.Marshal(resp)

	var gotBody models.OriginCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/origins" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.CreateOrigin(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateOrigin() unexpected error: %v", err)
	}
	if got.ID != "new-id" {
		t.Errorf("ID = %q, want %q", got.ID, "new-id")
	}
	if gotBody.HostName != req.HostName {
		t.Errorf("request body HostName = %q, want %q", gotBody.HostName, req.HostName)
	}
}

// ---- UpdateOrigin ----

func TestUpdateOrigin_Success(t *testing.T) {
	updated := models.Origin{ID: "o1", GroupID: "g2"}
	payload, _ := json.Marshal(updated)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/origins/o1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.UpdateOrigin(context.Background(), "o1", models.OriginUpdateRequest{GroupID: "g2"})
	if err != nil {
		t.Fatalf("UpdateOrigin() unexpected error: %v", err)
	}
	if got.GroupID != "g2" {
		t.Errorf("GroupID = %q, want %q", got.GroupID, "g2")
	}
}

// TestUpdateOrigin_202EmptyBody_ImmediateReady verifies that a 202 with an empty
// body triggers the polling loop and returns as soon as the origin reports READY.
func TestUpdateOrigin_202EmptyBody_ImmediateReady(t *testing.T) {
	ready := models.Origin{ID: "o1", GroupID: "g2", Status: "READY"}
	readyPayload, _ := json.Marshal(ready)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusAccepted) // 202 empty body
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(readyPayload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.UpdateOrigin(context.Background(), "o1", models.OriginUpdateRequest{GroupID: "g2"})
	if err != nil {
		t.Fatalf("UpdateOrigin() unexpected error: %v", err)
	}
	if got.Status != "READY" {
		t.Errorf("Status = %q, want READY", got.Status)
	}
}

// TestUpdateOrigin_202EmptyBody_GetError verifies that a non-retryable GET error
// during the polling loop is propagated to the caller.
func TestUpdateOrigin_202EmptyBody_GetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusAccepted) // 202 empty body
			return
		}
		// 400 is non-retryable, so the error surfaces immediately.
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.UpdateOrigin(context.Background(), "o1", models.OriginUpdateRequest{})
	if err == nil {
		t.Fatal("expected error from polling GET, got nil")
	}
}

// TestUpdateOrigin_202EmptyBody_Timeout verifies that the polling loop stops and
// returns an error when the context deadline expires before READY is observed.
// A short context timeout is used so the select hits Done() before the 5-second
// poll interval, keeping the test fast.
func TestUpdateOrigin_202EmptyBody_Timeout(t *testing.T) {
	updating := models.Origin{ID: "o1", Status: "UPDATING"}
	updatingPayload, _ := json.Marshal(updating)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusAccepted) // 202 empty body
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(updatingPayload)
	}))
	defer srv.Close()

	// 50 ms is much shorter than the 5-second poll interval, so the context
	// deadline fires in the select before time.After(originPollInterval).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(t, srv)
	_, err := c.UpdateOrigin(ctx, "o1", models.OriginUpdateRequest{})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want a timeout-related error", err.Error())
	}
}

// ---- DeleteOrigin ----

func TestDeleteOrigin_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/origins/o1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteOrigin(context.Background(), "o1"); err != nil {
		t.Fatalf("DeleteOrigin() unexpected error: %v", err)
	}
}

func TestDeleteOrigin_NotFoundIsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	// DeleteOrigin treats 404 as success (idempotent delete).
	if err := c.DeleteOrigin(context.Background(), "missing"); err != nil {
		t.Fatalf("DeleteOrigin() expected nil for 404, got: %v", err)
	}
}
