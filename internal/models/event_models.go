package models

import "time"

// Event represents an MSL5 Event resource as returned by the API.
type Event struct {
	EventID            string     `json:"event_id"`
	EventName          string     `json:"event_name"`
	StreamID           string     `json:"stream_id"`
	Active             bool       `json:"active"`
	Ended              bool       `json:"ended"`
	StartTime          *time.Time `json:"start_time,omitempty"`
	EndTime            *time.Time `json:"end_time,omitempty"`
	SubsegmentClipping *bool      `json:"subsegment_clipping,omitempty"`
	EgressFilepaths    []string   `json:"egress_filepaths,omitempty"`
	CreatedAt          time.Time  `json:"created_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at,omitempty"`
}

// EventCreateRequest is the payload for POST /api/v1/streams/{stream_id}/events.
type EventCreateRequest struct {
	EventName          string     `json:"event_name"`
	SourceEventName    *string    `json:"source_event_name,omitempty"`
	StartTime          *time.Time `json:"start_time,omitempty"`
	EndTime            *time.Time `json:"end_time,omitempty"`
	SubsegmentClipping *bool      `json:"subsegment_clipping,omitempty"`
}

// ListEventsResponse wraps the top-level list response.
type ListEventsResponse struct {
	Events []Event `json:"events"`
}
