package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
)

// ---- ListIngestCredentials ----

func TestListIngestCredentials_Success(t *testing.T) {
	desc := "test cred"
	creds := []models.IngestCredential{
		{ID: "cred-1", Username: "user1", Algorithm: models.HashAlgorithmSHA256},
		{ID: "cred-2", Username: "user2", Description: &desc, Algorithm: models.HashAlgorithmSHA512},
	}
	payload, _ := json.Marshal(creds)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/streams/s1/ingest_credentials" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.ListIngestCredentials(context.Background(), "s1")
	if err != nil {
		t.Fatalf("ListIngestCredentials() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(creds) = %d, want 2", len(got))
	}
	if got[0].ID != "cred-1" {
		t.Errorf("creds[0].ID = %q, want %q", got[0].ID, "cred-1")
	}
}

// ---- GetIngestCredentialByID ----

func TestGetIngestCredentialByID_Found(t *testing.T) {
	creds := []models.IngestCredential{
		{ID: "cred-1", Username: "user1", Algorithm: models.HashAlgorithmMD5},
		{ID: "cred-2", Username: "user2", Algorithm: models.HashAlgorithmSHA256},
	}
	payload, _ := json.Marshal(creds)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetIngestCredentialByID(context.Background(), "s1", "cred-2")
	if err != nil {
		t.Fatalf("GetIngestCredentialByID() unexpected error: %v", err)
	}
	if got.ID != "cred-2" {
		t.Errorf("ID = %q, want %q", got.ID, "cred-2")
	}
	if got.Username != "user2" {
		t.Errorf("Username = %q, want %q", got.Username, "user2")
	}
}

func TestGetIngestCredentialByID_NotFound(t *testing.T) {
	// List returns creds, but none match the requested ID.
	creds := []models.IngestCredential{
		{ID: "cred-1", Username: "user1", Algorithm: models.HashAlgorithmSHA256},
	}
	payload, _ := json.Marshal(creds)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.GetIngestCredentialByID(context.Background(), "s1", "does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound() = false, want true; err = %v", err)
	}
}

// ---- CreateIngestCredential ----

func TestCreateIngestCredential_Success(t *testing.T) {
	req := models.IngestCredentialCreateRequest{
		Username:  "newuser",
		Password:  "s3cret",
		Algorithm: models.HashAlgorithmSHA512256,
	}
	resp := models.IngestCredential{ID: "cred-new", Username: "newuser", Algorithm: models.HashAlgorithmSHA512256}
	payload, _ := json.Marshal(resp)

	var gotBody models.IngestCredentialCreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/streams/s1/ingest_credentials" {
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
	got, err := c.CreateIngestCredential(context.Background(), "s1", req)
	if err != nil {
		t.Fatalf("CreateIngestCredential() unexpected error: %v", err)
	}
	if got.ID != "cred-new" {
		t.Errorf("ID = %q, want %q", got.ID, "cred-new")
	}
	if gotBody.Username != req.Username {
		t.Errorf("request body Username = %q, want %q", gotBody.Username, req.Username)
	}
	if gotBody.Algorithm != req.Algorithm {
		t.Errorf("request body Algorithm = %q, want %q", gotBody.Algorithm, req.Algorithm)
	}
}

// ---- UpdateIngestCredential ----

func TestUpdateIngestCredential_Success(t *testing.T) {
	desc := "updated description"
	req := models.IngestCredentialUpdateRequest{Description: &desc}
	resp := models.IngestCredential{ID: "cred-1", Username: "user1", Description: &desc, Algorithm: models.HashAlgorithmSHA256}
	payload, _ := json.Marshal(resp)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/streams/s1/ingest_credentials/cred-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.UpdateIngestCredential(context.Background(), "s1", "cred-1", req)
	if err != nil {
		t.Fatalf("UpdateIngestCredential() unexpected error: %v", err)
	}
	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
}

// ---- DeleteIngestCredential ----

func TestDeleteIngestCredential_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/v1/streams/s1/ingest_credentials/cred-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteIngestCredential(context.Background(), "s1", "cred-1"); err != nil {
		t.Fatalf("DeleteIngestCredential() unexpected error: %v", err)
	}
}

func TestDeleteIngestCredential_NotFoundIsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`not found`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.DeleteIngestCredential(context.Background(), "s1", "gone-cred"); err != nil {
		t.Fatalf("DeleteIngestCredential() expected nil for 404, got: %v", err)
	}
}
