package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

// ---- ListEvents ----

func TestListEvents_WrappedResponse(t *testing.T) {
	events := []models.Event{
		{EventID: "e1", EventName: "event-one", StreamID: "s1"},
		{EventID: "e2", EventName: "event-two", StreamID: "s1"},
	}
	payload, _ := json.Marshal(models.ListEventsResponse{Events: events})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/streams/s1/events" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListEvents(context.Background(), "s1")
	if err != nil {
		t.Fatalf("ListEvents() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(got))
	}
	if got[0].EventID != "e1" {
		t.Errorf("events[0].EventID = %q, want %q", got[0].EventID, "e1")
	}
}

func TestListEvents_DirectArrayResponse(t *testing.T) {
	events := []models.Event{{EventID: "e1", EventName: "only-event", StreamID: "s1"}}
	payload, _ := json.Marshal(events)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListEvents(context.Background(), "s1")
	if err != nil {
		t.Fatalf("ListEvents() unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].EventID != "e1" {
		t.Errorf("unexpected result: %+v", got)
	}
}

// ---- GetEvent ----

func TestGetEvent_Success(t *testing.T) {
	event := models.Event{EventID: "e1", EventName: "my-event", StreamID: "s1"}
	payload, _ := json.Marshal(event)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/streams/s1/events/my-event" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetEvent(context.Background(), "s1", "my-event")
	if err != nil {
		t.Fatalf("GetEvent() unexpected error: %v", err)
	}
	if got.EventName != "my-event" {
		t.Errorf("EventName = %q, want %q", got.EventName, "my-event")
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetEvent(context.Background(), "s1", "missing-event")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound() = false, want true; err = %v", err)
	}
}

// ---- CreateEvent ----

func TestCreateEvent_Success(t *testing.T) {
	req := models.EventCreateRequest{EventName: "new-event"}
	resp := models.Event{EventID: "e-new", EventName: "new-event", StreamID: "s1"}
	payload, _ := json.Marshal(resp)

	var gotBody models.EventCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/streams/s1/events" {
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
	got, err := c.CreateEvent(context.Background(), "s1", req)
	if err != nil {
		t.Fatalf("CreateEvent() unexpected error: %v", err)
	}
	if got.EventID != "e-new" {
		t.Errorf("EventID = %q, want %q", got.EventID, "e-new")
	}
	if gotBody.EventName != req.EventName {
		t.Errorf("request body EventName = %q, want %q", gotBody.EventName, req.EventName)
	}
}

// ---- DeleteEvent ----

func TestDeleteEvent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/streams/s1/events/my-event" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteEvent(context.Background(), "s1", "my-event"); err != nil {
		t.Fatalf("DeleteEvent() unexpected error: %v", err)
	}
}

func TestDeleteEvent_NotFoundIsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteEvent(context.Background(), "s1", "gone-event"); err != nil {
		t.Fatalf("DeleteEvent() expected nil for 404, got: %v", err)
	}
}
