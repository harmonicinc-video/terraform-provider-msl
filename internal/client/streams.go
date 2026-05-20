package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

const (
	streamPollInterval = 5 * time.Second
	streamPollTimeout  = 5 * time.Minute
)

// ListStreams returns all streams, optionally filtered by contractID and groupID.
func (c *Client) ListStreams(ctx context.Context, contractID, groupID string) ([]models.Stream, error) {
	path := "/api/v1/streams"
	params := url.Values{}
	if contractID != "" {
		params.Set("contract_id", contractID)
	}
	if groupID != "" {
		params.Set("group_id", groupID)
	}
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}

	body, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing streams: %w", err)
	}

	var resp models.ListStreamsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// Fallback: API may return a bare array
		var streams []models.Stream
		if err2 := json.Unmarshal(body, &streams); err2 != nil {
			return nil, fmt.Errorf("parsing list streams response: %w", err)
		}
		return streams, nil
	}
	return resp.Streams, nil
}

// GetStream retrieves a single stream by its ID.
// Returns an *APIError with StatusCode 404 if the stream does not exist.
func (c *Client) GetStream(ctx context.Context, id string) (*models.Stream, error) {
	body, _, err := c.do(ctx, http.MethodGet, "/api/v1/streams/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting stream %s: %w", id, err)
	}

	var stream models.Stream
	if err := json.Unmarshal(body, &stream); err != nil {
		return nil, fmt.Errorf("parsing get stream response: %w", err)
	}
	return &stream, nil
}

// CreateStream creates a new stream and polls until it reaches READY status.
// The API returns 202 Accepted; this method blocks until the stream is READY
// or the context is cancelled (5-minute default timeout).
func (c *Client) CreateStream(ctx context.Context, req models.StreamCreateRequest) (*models.Stream, error) {
	body, statusCode, err := c.do(ctx, http.MethodPost, "/api/v1/streams", req)
	if err != nil {
		return nil, fmt.Errorf("creating stream: %w", err)
	}

	// Parse the ID from the initial response (may be 200/201 or 202 with body)
	var stream models.Stream
	if len(body) > 0 {
		if err := json.Unmarshal(body, &stream); err != nil {
			return nil, fmt.Errorf("parsing create stream response: %w", err)
		}
	}

	if statusCode == http.StatusAccepted || stream.Status == models.StreamStatusCreating {
		if stream.ID == "" {
			return nil, fmt.Errorf("create stream: API returned 202 but no stream_id in response body")
		}
		return c.pollStreamUntilReady(ctx, stream.ID)
	}

	return &stream, nil
}

// UpdateStream updates mutable fields on a stream.
// The API uses full-replace semantics: omitted optional fields reset to defaults.
// Returns 202 Accepted; this method re-fetches the final state.
func (c *Client) UpdateStream(ctx context.Context, id string, req models.StreamUpdateRequest) (*models.Stream, error) {
	body, _, err := c.do(ctx, http.MethodPut, "/api/v1/streams/"+id, req)
	if err != nil {
		return nil, fmt.Errorf("updating stream %s: %w", id, err)
	}

	if len(body) == 0 {
		// 202 with empty body — poll for final state
		return c.pollStreamUntilReady(ctx, id)
	}

	var stream models.Stream
	if err := json.Unmarshal(body, &stream); err != nil {
		return nil, fmt.Errorf("parsing update stream response: %w", err)
	}
	return &stream, nil
}

// DeleteStream deletes a stream by ID.
// Returns nil if the stream was already deleted (404).
func (c *Client) DeleteStream(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/api/v1/streams/"+id, nil)
	if err != nil {
		if IsNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("deleting stream %s: %w", id, err)
	}
	return nil
}

// pollStreamUntilReady polls GetStream every 5 seconds until status is READY,
// or until the context deadline (max 5 minutes from call time).
func (c *Client) pollStreamUntilReady(ctx context.Context, id string) (*models.Stream, error) {
	deadline := time.Now().Add(streamPollTimeout)
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if c.debugLog {
		log.Printf("[DEBUG] msl: polling stream %s until READY (timeout %s)", id, streamPollTimeout)
	}
	attempt := 0
	for {
		attempt++
		stream, err := c.GetStream(pollCtx, id)
		if err != nil {
			return nil, fmt.Errorf("polling stream %s: %w", id, err)
		}
		if c.debugLog {
			log.Printf("[DEBUG] msl: poll attempt %d for stream %s: status=%s", attempt, id, stream.Status)
		}
		if stream.Status == models.StreamStatusReady {
			return stream, nil
		}
		if stream.Status == models.StreamStatusDeleted {
			return nil, fmt.Errorf("stream %s was deleted while waiting for READY status", id)
		}

		select {
		case <-time.After(streamPollInterval):
		case <-pollCtx.Done():
			return nil, fmt.Errorf("timed out waiting for stream %s to reach READY status (last status: %s)", id, stream.Status)
		}
	}
}
