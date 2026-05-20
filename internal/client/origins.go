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
	originPollInterval = 5 * time.Second
	originPollTimeout  = 5 * time.Minute
)

// ListOrigins returns all origins, optionally filtered by contractID and groupID.
func (c *Client) ListOrigins(ctx context.Context, contractID, groupID string) ([]models.Origin, error) {
	path := "/api/v1/origins"
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
		return nil, fmt.Errorf("listing origins: %w", err)
	}

	var resp models.ListOriginsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// Some APIs return the array directly without a wrapper
		var origins []models.Origin
		if err2 := json.Unmarshal(body, &origins); err2 != nil {
			return nil, fmt.Errorf("parsing list origins response: %w", err)
		}
		return origins, nil
	}
	return resp.Origins, nil
}

// CreateOrigin creates a new origin and returns the created resource.
func (c *Client) CreateOrigin(ctx context.Context, req models.OriginCreateRequest) (*models.Origin, error) {
	body, _, err := c.do(ctx, http.MethodPost, "/api/v1/origins", req)
	if err != nil {
		return nil, fmt.Errorf("creating origin: %w", err)
	}

	var origin models.Origin
	if err := json.Unmarshal(body, &origin); err != nil {
		return nil, fmt.Errorf("parsing create origin response: %w", err)
	}
	return &origin, nil
}

// GetOrigin retrieves a single origin by its ID.
// Returns an *APIError with StatusCode 404 if the origin does not exist.
func (c *Client) GetOrigin(ctx context.Context, id string) (*models.Origin, error) {
	body, _, err := c.do(ctx, http.MethodGet, "/api/v1/origins/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("getting origin %s: %w", id, err)
	}

	var origin models.Origin
	if err := json.Unmarshal(body, &origin); err != nil {
		return nil, fmt.Errorf("parsing get origin response: %w", err)
	}
	return &origin, nil
}

// UpdateOrigin updates the mutable fields (group_id, shared_keys) on an origin.
// If the API returns 202 Accepted with an empty body, this method polls GetOrigin
// every 5 seconds until the origin status is "READY" or the 5-minute timeout elapses.
func (c *Client) UpdateOrigin(ctx context.Context, id string, req models.OriginUpdateRequest) (*models.Origin, error) {
	body, _, err := c.do(ctx, http.MethodPut, "/api/v1/origins/"+id, req)
	if err != nil {
		return nil, fmt.Errorf("updating origin %s: %w", id, err)
	}

	if len(body) == 0 {
		// 202 with empty body — poll for final state
		return c.pollOriginUntilReady(ctx, id)
	}

	var origin models.Origin
	if err := json.Unmarshal(body, &origin); err != nil {
		return nil, fmt.Errorf("parsing update origin response: %w", err)
	}
	return &origin, nil
}

// pollOriginUntilReady polls GetOrigin every 5 seconds until status is READY,
// or until the context deadline (max 5 minutes from call time).
func (c *Client) pollOriginUntilReady(ctx context.Context, id string) (*models.Origin, error) {
	deadline := time.Now().Add(originPollTimeout)
	pollCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	if c.debugLog {
		log.Printf("[DEBUG] msl: polling origin %s until READY (timeout %s)", id, originPollTimeout)
	}
	attempt := 0
	for {
		attempt++
		origin, err := c.GetOrigin(pollCtx, id)
		if err != nil {
			return nil, fmt.Errorf("polling origin %s: %w", id, err)
		}
		if c.debugLog {
			log.Printf("[DEBUG] msl: poll attempt %d for origin %s: status=%s", attempt, id, origin.Status)
		}
		if origin.Status == models.OriginStatusReady {
			return origin, nil
		}

		select {
		case <-time.After(originPollInterval):
		case <-pollCtx.Done():
			return nil, fmt.Errorf("timed out waiting for origin %s to reach READY status (last status: %s)", id, origin.Status)
		}
	}
}

// DeleteOrigin deletes an origin by ID.
// Returns nil if the origin was already deleted (404).
func (c *Client) DeleteOrigin(ctx context.Context, id string) error {
	_, _, err := c.do(ctx, http.MethodDelete, "/api/v1/origins/"+id, nil)
	if err != nil {
		if IsNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("deleting origin %s: %w", id, err)
	}
	return nil
}
