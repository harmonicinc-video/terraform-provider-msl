package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

// ListIngestCredentials returns all ingest credentials for a given stream.
func (c *Client) ListIngestCredentials(ctx context.Context, streamID string) ([]models.IngestCredential, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/ingest_credentials", streamID)

	body, _, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing ingest credentials for stream %s: %w", streamID, err)
	}

	var creds []models.IngestCredential
	if err := json.Unmarshal(body, &creds); err != nil {
		return nil, fmt.Errorf("parsing list ingest credentials response: %w", err)
	}
	return creds, nil
}

// CreateIngestCredential creates a new ingest credential for a stream.
func (c *Client) CreateIngestCredential(ctx context.Context, streamID string, req models.IngestCredentialCreateRequest) (*models.IngestCredential, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/ingest_credentials", streamID)

	body, _, err := c.do(ctx, http.MethodPost, path, req)
	if err != nil {
		return nil, fmt.Errorf("creating ingest credential for stream %s: %w", streamID, err)
	}

	var cred models.IngestCredential
	if err := json.Unmarshal(body, &cred); err != nil {
		return nil, fmt.Errorf("parsing create ingest credential response: %w", err)
	}
	return &cred, nil
}

// GetIngestCredentialByID retrieves a single ingest credential by listing all credentials
// for the stream and finding the one matching credID.
// There is no GET-by-ID endpoint in the API; this uses list-and-find.
// Returns an *APIError with StatusCode 404 if the credential is not found.
func (c *Client) GetIngestCredentialByID(ctx context.Context, streamID, credID string) (*models.IngestCredential, error) {
	creds, err := c.ListIngestCredentials(ctx, streamID)
	if err != nil {
		return nil, fmt.Errorf("fetching ingest credential %s: %w", credID, err)
	}

	for i := range creds {
		if creds[i].ID == credID {
			return &creds[i], nil
		}
	}

	// Not found — synthesise a 404 APIError to match the pattern used by other resources
	return nil, &APIError{StatusCode: http.StatusNotFound, Body: fmt.Sprintf("ingest credential %s not found in stream %s", credID, streamID)}
}

// UpdateIngestCredential updates the description and/or expiry_date of an ingest credential.
func (c *Client) UpdateIngestCredential(ctx context.Context, streamID, credID string, req models.IngestCredentialUpdateRequest) (*models.IngestCredential, error) {
	path := fmt.Sprintf("/api/v1/streams/%s/ingest_credentials/%s", streamID, credID)

	body, _, err := c.do(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, fmt.Errorf("updating ingest credential %s in stream %s: %w", credID, streamID, err)
	}

	var cred models.IngestCredential
	if err := json.Unmarshal(body, &cred); err != nil {
		return nil, fmt.Errorf("parsing update ingest credential response: %w", err)
	}
	return &cred, nil
}

// DeleteIngestCredential deletes an ingest credential.
// The API returns 204 No Content on success.
// Returns nil if the credential was already deleted (404).
func (c *Client) DeleteIngestCredential(ctx context.Context, streamID, credID string) error {
	path := fmt.Sprintf("/api/v1/streams/%s/ingest_credentials/%s", streamID, credID)

	_, _, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		if IsNotFound(err) {
			return nil // already gone
		}
		return fmt.Errorf("deleting ingest credential %s in stream %s: %w", credID, streamID, err)
	}
	return nil
}
