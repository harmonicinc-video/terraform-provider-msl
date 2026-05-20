package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

// ListEvents returns all events for a given stream.
func (c *Client) ListEvents(ctx context.Context, streamID string) ([]models.Event, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/events", streamID)

	body, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing events for stream %s: %w", streamID, err)
	}

	var resp models.ListEventsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// Fallback: API may return a bare array
		var events []models.Event
		if err2 := json.Unmarshal(body, &events); err2 != nil {
			return nil, fmt.Errorf("parsing list events response: %w", err)
		}
		return events, nil
	}
	return resp.Events, nil
}

// CreateEvent creates a new event within a stream.
// The API returns 201 Created with the event in the response body.
func (c *Client) CreateEvent(ctx context.Context, streamID string, req models.EventCreateRequest) (*models.Event, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/events", streamID)

	body, _, err := c.do(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("creating event in stream %s: %w", streamID, err)
	}

	var event models.Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("parsing create event response: %w", err)
	}
	return &event, nil
}

// GetEvent retrieves a single event by stream ID and event name.
// The API path uses event_name (not event_id) as the identifier.
// Returns an *APIError with StatusCode 404 if the event does not exist.
func (c *Client) GetEvent(ctx context.Context, streamID, eventName string) (*models.Event, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/events/%s", streamID, eventName)

	body, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting event %s in stream %s: %w", eventName, streamID, err)
	}

	var event models.Event
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("parsing get event response: %w", err)
	}
	return &event, nil
}

// DeleteEvent deletes an event by stream ID and event name.
// Returns nil if the event was already deleted (404).
func (c *Client) DeleteEvent(ctx context.Context, streamID, eventName string) error {
	path := fmt.Sprintf("/api/v1/streams/%s/events/%s", streamID, eventName)

	_, _, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		if IsNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("deleting event %s in stream %s: %w", eventName, streamID, err)
	}
	return nil
}
