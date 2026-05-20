package models

import (
	"encoding/json"
	"testing"
	"time"
)

// ---- Event JSON round-trip ----

func TestEvent_JSONRoundTrip(t *testing.T) {
	now := time.Date(2024, 3, 10, 12, 0, 0, 0, time.UTC)
	start := time.Date(2024, 3, 10, 14, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 10, 16, 0, 0, 0, time.UTC)
	clipping := true
	orig := Event{
		EventID:            "evt-1",
		EventName:          "my-event",
		StreamID:           "stream-1",
		Active:             true,
		Ended:              false,
		StartTime:          &start,
		EndTime:            &end,
		SubsegmentClipping: &clipping,
		EgressFilepaths:    []string{"/path/a", "/path/b"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.EventID != orig.EventID {
		t.Errorf("EventID = %q, want %q", got.EventID, orig.EventID)
	}
	if got.EventName != orig.EventName {
		t.Errorf("EventName = %q, want %q", got.EventName, orig.EventName)
	}
	if got.StreamID != orig.StreamID {
		t.Errorf("StreamID = %q, want %q", got.StreamID, orig.StreamID)
	}
	if got.Active != orig.Active {
		t.Errorf("Active = %v, want %v", got.Active, orig.Active)
	}
	if got.Ended != orig.Ended {
		t.Errorf("Ended = %v, want %v", got.Ended, orig.Ended)
	}
	if got.StartTime == nil || !got.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", got.StartTime, start)
	}
	if got.EndTime == nil || !got.EndTime.Equal(end) {
		t.Errorf("EndTime = %v, want %v", got.EndTime, end)
	}
	if got.SubsegmentClipping == nil || *got.SubsegmentClipping != clipping {
		t.Errorf("SubsegmentClipping = %v, want %v", got.SubsegmentClipping, clipping)
	}
	if len(got.EgressFilepaths) != 2 {
		t.Fatalf("len(EgressFilepaths) = %d, want 2", len(got.EgressFilepaths))
	}
	if got.EgressFilepaths[0] != "/path/a" {
		t.Errorf("EgressFilepaths[0] = %q, want %q", got.EgressFilepaths[0], "/path/a")
	}
}

func TestEvent_JSONFieldNames(t *testing.T) {
	evt := Event{EventID: "e1", EventName: "n1", StreamID: "s1"}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, want := range []string{"event_id", "event_name", "stream_id", "active", "ended"} {
		if _, ok := m[want]; !ok {
			t.Errorf("JSON key %q missing from marshalled Event", want)
		}
	}
}

func TestEvent_OptionalFieldsOmitted(t *testing.T) {
	evt := Event{EventID: "e1", EventName: "n1", StreamID: "s1"}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"start_time", "end_time", "subsegment_clipping", "egress_filepaths"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil/empty", absent)
		}
	}
}

// ---- EventCreateRequest ----

func TestEventCreateRequest_JSONRoundTrip(t *testing.T) {
	start := time.Date(2024, 5, 1, 8, 0, 0, 0, time.UTC)
	end := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC)
	clipping := false
	src := "source-event"
	req := EventCreateRequest{
		EventName:          "live-event",
		SourceEventName:    &src,
		StartTime:          &start,
		EndTime:            &end,
		SubsegmentClipping: &clipping,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got EventCreateRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.EventName != req.EventName {
		t.Errorf("EventName = %q, want %q", got.EventName, req.EventName)
	}
	if got.SourceEventName == nil || *got.SourceEventName != src {
		t.Errorf("SourceEventName = %v, want %q", got.SourceEventName, src)
	}
	if got.StartTime == nil || !got.StartTime.Equal(start) {
		t.Errorf("StartTime = %v, want %v", got.StartTime, start)
	}
	if got.SubsegmentClipping == nil || *got.SubsegmentClipping != clipping {
		t.Errorf("SubsegmentClipping = %v, want %v", got.SubsegmentClipping, clipping)
	}
}

func TestEventCreateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := EventCreateRequest{EventName: "minimal-event"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"source_event_name", "start_time", "end_time", "subsegment_clipping"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil", absent)
		}
	}
}

// ---- ListEventsResponse ----

func TestListEventsResponse_Unmarshal(t *testing.T) {
	raw := `{"events":[{"event_id":"e1","event_name":"n1","stream_id":"s1","active":true,"ended":false},{"event_id":"e2","event_name":"n2","stream_id":"s1","active":false,"ended":true}]}`
	var resp ListEventsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(resp.Events) != 2 {
		t.Fatalf("len(Events) = %d, want 2", len(resp.Events))
	}
	if resp.Events[0].EventID != "e1" {
		t.Errorf("Events[0].EventID = %q, want %q", resp.Events[0].EventID, "e1")
	}
	if !resp.Events[1].Ended {
		t.Errorf("Events[1].Ended = false, want true")
	}
}

func TestListEventsResponse_EmptyList(t *testing.T) {
	raw := `{"events":[]}`
	var resp ListEventsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(resp.Events) != 0 {
		t.Errorf("len(Events) = %d, want 0", len(resp.Events))
	}
}
