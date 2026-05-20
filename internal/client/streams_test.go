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

// ---- ListStreams ----

func TestListStreams_WrappedResponse(t *testing.T) {
	streams := []models.Stream{
		{ID: "s1", Format: models.StreamFormatHLS, ContractID: "c1"},
		{ID: "s2", Format: models.StreamFormatCMAF, ContractID: "c1"},
	}
	payload, _ := json.Marshal(models.ListStreamsResponse{Streams: streams})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/streams" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListStreams(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListStreams() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(streams) = %d, want 2", len(got))
	}
	if got[0].ID != "s1" {
		t.Errorf("streams[0].ID = %q, want %q", got[0].ID, "s1")
	}
}

func TestListStreams_DirectArrayResponse(t *testing.T) {
	streams := []models.Stream{{ID: "s1", Format: models.StreamFormatDASH}}
	payload, _ := json.Marshal(streams)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListStreams(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListStreams() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != "s1" {
		t.Errorf("unexpected result: %+v", got)
	}
}

func TestListStreams_ForwardsQueryParams(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"streams":[]}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.ListStreams(context.Background(), "ctr1", "grp1")
	if err != nil {
		t.Fatalf("ListStreams() unexpected error: %v", err)
	}
	if gotQuery == "" {
		t.Error("expected query params to be forwarded, got empty string")
	}
}

// ---- GetStream ----

func TestGetStream_Success(t *testing.T) {
	stream := models.Stream{ID: "s1", Format: models.StreamFormatHLS, Status: models.StreamStatusReady}
	payload, _ := json.Marshal(stream)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/streams/s1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetStream(context.Background(), "s1")
	if err != nil {
		t.Fatalf("GetStream() unexpected error: %v", err)
	}
	if got.ID != "s1" {
		t.Errorf("ID = %q, want %q", got.ID, "s1")
	}
}

func TestGetStream_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetStream(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound() = false, want true; err = %v", err)
	}
}

// ---- CreateStream ----

func TestCreateStream_ImmediateReady(t *testing.T) {
	// 201 with a READY stream body — no polling required.
	stream := models.Stream{ID: "s-new", Format: models.StreamFormatHLS, Status: models.StreamStatusReady}
	payload, _ := json.Marshal(stream)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/streams" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.CreateStream(context.Background(), models.StreamCreateRequest{
		Format:     models.StreamFormatHLS,
		OriginID:   "o1",
		ContractID: "c1",
		CPTag:      "tag1",
		GroupID:    "g1",
		Archiving:  models.Archiving{NoArchive: true},
	})
	if err != nil {
		t.Fatalf("CreateStream() unexpected error: %v", err)
	}
	if got.ID != "s-new" {
		t.Errorf("ID = %q, want %q", got.ID, "s-new")
	}
}

// TestCreateStream_202PollsUntilReady verifies that a 202 Accepted response
// triggers the polling loop and returns once the stream reports READY.
func TestCreateStream_202PollsUntilReady(t *testing.T) {
	initial := models.Stream{ID: "s-new", Status: models.StreamStatusCreating}
	initialPayload, _ := json.Marshal(initial)
	ready := models.Stream{ID: "s-new", Status: models.StreamStatusReady}
	readyPayload, _ := json.Marshal(ready)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			// 202 with stream ID so polling can begin
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(initialPayload)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(readyPayload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.CreateStream(context.Background(), models.StreamCreateRequest{
		Format: models.StreamFormatHLS, OriginID: "o1", ContractID: "c1",
		CPTag: "tag1", GroupID: "g1", Archiving: models.Archiving{NoArchive: true},
	})
	if err != nil {
		t.Fatalf("CreateStream() unexpected error: %v", err)
	}
	if got.Status != models.StreamStatusReady {
		t.Errorf("Status = %q, want READY", got.Status)
	}
}

// TestCreateStream_202GetError verifies that a non-retryable GET error during
// the polling loop is propagated to the caller.
func TestCreateStream_202GetError(t *testing.T) {
	initial := models.Stream{ID: "s-new", Status: models.StreamStatusCreating}
	initialPayload, _ := json.Marshal(initial)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(initialPayload)
			return
		}
		// 400 is non-retryable so the error surfaces immediately.
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.CreateStream(context.Background(), models.StreamCreateRequest{
		Format: models.StreamFormatHLS, OriginID: "o1", ContractID: "c1",
		CPTag: "tag1", GroupID: "g1", Archiving: models.Archiving{NoArchive: true},
	})
	if err == nil {
		t.Fatal("expected error from polling GET, got nil")
	}
}

// TestCreateStream_202Timeout verifies that the polling loop stops and returns
// an error when the context deadline expires before READY is observed.
func TestCreateStream_202Timeout(t *testing.T) {
	initial := models.Stream{ID: "s-new", Status: models.StreamStatusCreating}
	initialPayload, _ := json.Marshal(initial)
	creating := models.Stream{ID: "s-new", Status: models.StreamStatusCreating}
	creatingPayload, _ := json.Marshal(creating)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(initialPayload)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(creatingPayload)
	}))
	defer srv.Close()

	// 50 ms is much shorter than the 5-second poll interval, so the context
	// deadline fires in the select before time.After(streamPollInterval).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(t, srv)
	_, err := c.CreateStream(ctx, models.StreamCreateRequest{
		Format: models.StreamFormatHLS, OriginID: "o1", ContractID: "c1",
		CPTag: "tag1", GroupID: "g1", Archiving: models.Archiving{NoArchive: true},
	})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want a timeout-related error", err.Error())
	}
}

// ---- UpdateStream ----

func TestUpdateStream_WithBody(t *testing.T) {
	groupID := "g2"
	updated := models.Stream{ID: "s1", GroupID: "g2", Status: models.StreamStatusReady}
	payload, _ := json.Marshal(updated)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/streams/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.UpdateStream(context.Background(), "s1", models.StreamUpdateRequest{GroupID: &groupID})
	if err != nil {
		t.Fatalf("UpdateStream() unexpected error: %v", err)
	}
	if got.GroupID != "g2" {
		t.Errorf("GroupID = %q, want %q", got.GroupID, "g2")
	}
}

// TestUpdateStream_202PollsUntilReady verifies that a 202 with empty body
// triggers the polling loop and returns once the stream reports READY.
func TestUpdateStream_202PollsUntilReady(t *testing.T) {
	ready := models.Stream{ID: "s1", GroupID: "g2", Status: models.StreamStatusReady}
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

	groupID := "g2"
	c := newTestClient(t, srv)
	got, err := c.UpdateStream(context.Background(), "s1", models.StreamUpdateRequest{GroupID: &groupID})
	if err != nil {
		t.Fatalf("UpdateStream() unexpected error: %v", err)
	}
	if got.Status != models.StreamStatusReady {
		t.Errorf("Status = %q, want READY", got.Status)
	}
}

// TestUpdateStream_202GetError verifies that a non-retryable GET error during
// the polling loop is propagated to the caller.
func TestUpdateStream_202GetError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusAccepted) // 202 empty body
			return
		}
		// 400 is non-retryable so the error surfaces immediately.
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))
	defer srv.Close()

	groupID := "g2"
	c := newTestClient(t, srv)
	_, err := c.UpdateStream(context.Background(), "s1", models.StreamUpdateRequest{GroupID: &groupID})
	if err == nil {
		t.Fatal("expected error from polling GET, got nil")
	}
}

// TestUpdateStream_202Timeout verifies that the polling loop stops and returns
// an error when the context deadline expires before READY is observed.
func TestUpdateStream_202Timeout(t *testing.T) {
	updating := models.Stream{ID: "s1", Status: "UPDATING"}
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
	// deadline fires in the select before time.After(streamPollInterval).
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	groupID := "g2"
	c := newTestClient(t, srv)
	_, err := c.UpdateStream(ctx, "s1", models.StreamUpdateRequest{GroupID: &groupID})
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "context") {
		t.Errorf("error = %q, want a timeout-related error", err.Error())
	}
}

// ---- DeleteStream ----

func TestDeleteStream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/streams/s1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteStream(context.Background(), "s1"); err != nil {
		t.Fatalf("DeleteStream() unexpected error: %v", err)
	}
}

func TestDeleteStream_NotFoundIsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteStream(context.Background(), "missing"); err != nil {
		t.Fatalf("DeleteStream() expected nil for 404, got: %v", err)
	}
}
