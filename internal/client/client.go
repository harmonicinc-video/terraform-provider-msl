// Package client provides an HTTP client for interacting with the MSL API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultMaxRetries = 3
	backoffBase       = 1 * time.Second
	backoffMax        = 30 * time.Second
)

// Client is the MSL5 API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	debugLog   bool
	// token is never logged — kept unexported and not included in any fmt output.
	token string
}

// Config holds configuration for constructing a Client.
type Config struct {
	BaseURL        string
	Token          string
	RequestTimeout time.Duration
	MaxRetries     int
	DebugLog       bool
}

// New creates a configured MSL5 API Client.
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("MSL5 API base URL must not be empty")
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("MSL5 API token must not be empty")
	}

	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = defaultMaxRetries
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		token:      cfg.Token,
		maxRetries: maxRetries,
		debugLog:   cfg.DebugLog,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// APIError represents a non-retryable API error response.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("MSL5 API error (HTTP %d): %s", e.StatusCode, e.Body)
}

// IsNotFound returns true when the API responded with HTTP 404.
// It traverses wrapped errors (errors created with %w).
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// do executes an HTTP request with retry/backoff logic.
// It attaches the Bearer token and never logs it.
// path may include a query string (e.g. "/api/v1/origins?contract_id=X").
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	baseU, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, 0, fmt.Errorf("building request URL: %w", err)
	}
	ref, err := url.Parse(path)
	if err != nil {
		return nil, 0, fmt.Errorf("building request URL: %w", err)
	}
	reqURL := baseU.ResolveReference(ref).String()

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			wait := backoffDuration(attempt)
			if c.debugLog {
				log.Printf("[DEBUG] msl: retry %d/%d after %s (last error: %v)", attempt, c.maxRetries, wait, lastErr)
			}
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			}
		}

		var bodyReader io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			if err != nil {
				return nil, 0, fmt.Errorf("marshalling request body: %w", err)
			}
			bodyReader = bytes.NewReader(encoded)
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
		if err != nil {
			return nil, 0, fmt.Errorf("creating HTTP request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token) // token never logged
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		if c.debugLog {
			log.Printf("[DEBUG] msl: %s %s", method, reqURL)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if isContextError(ctx) {
				return nil, 0, ctx.Err()
			}
			// Network/timeout error — retryable
			if c.debugLog {
				log.Printf("[DEBUG] msl: network error on %s %s (attempt %d/%d): %v", method, reqURL, attempt+1, c.maxRetries+1, err)
			}
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("reading response body: %w", readErr)
			continue
		}

		if c.debugLog {
			log.Printf("[DEBUG] msl: response HTTP %d, body length %d bytes", resp.StatusCode, len(respBody))
		}

		statusCode := resp.StatusCode

		// Success range
		if statusCode >= 200 && statusCode < 300 {
			return respBody, statusCode, nil
		}

		// Retryable: 5xx, 408 (Request Timeout), 429 (Too Many Requests)
		if statusCode >= 500 || statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests {
			lastErr = &APIError{StatusCode: statusCode, Body: string(respBody)}
			if c.debugLog {
				log.Printf("[DEBUG] msl: retryable error on %s %s (attempt %d/%d): HTTP %d", method, reqURL, attempt+1, c.maxRetries+1, statusCode)
			}
			continue
		}

		// Non-retryable 4xx
		if c.debugLog {
			log.Printf("[DEBUG] msl: non-retryable error on %s %s: HTTP %d", method, reqURL, statusCode)
		}
		return nil, statusCode, &APIError{StatusCode: statusCode, Body: string(respBody)}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("request failed after %d attempts", c.maxRetries+1)
	}
	return nil, 0, fmt.Errorf("MSL5 API request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// backoffDuration returns exponential backoff capped at backoffMax.
// The overflow-safe comparison is done in float64 before converting to time.Duration.
func backoffDuration(attempt int) time.Duration {
	exp := math.Pow(2, float64(attempt-1))
	if exp*float64(backoffBase) >= float64(backoffMax) {
		return backoffMax
	}
	return time.Duration(float64(backoffBase) * exp)
}

func isContextError(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
