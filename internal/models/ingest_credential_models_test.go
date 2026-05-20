package models

import (
	"encoding/json"
	"testing"
	"time"
)

// ---- HashAlgorithm constants ----

func TestHashAlgorithm_Values(t *testing.T) {
	cases := []struct {
		alg  HashAlgorithm
		want string
	}{
		{HashAlgorithmSHA256, "SHA256"},
		{HashAlgorithmSHA512256, "SHA512_256"},
		{HashAlgorithmSHA512, "SHA512"},
		{HashAlgorithmMD5, "MD5"},
	}
	for _, tc := range cases {
		if string(tc.alg) != tc.want {
			t.Errorf("HashAlgorithm value = %q, want %q", tc.alg, tc.want)
		}
	}
}

// ---- IngestCredential JSON round-trip ----

func TestIngestCredential_JSONRoundTrip(t *testing.T) {
	desc := "test credential"
	expiry := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	orig := IngestCredential{
		ID:          "cred-1",
		Username:    "user1",
		Description: &desc,
		Algorithm:   HashAlgorithmSHA256,
		ExpiryDate:  &expiry,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got IngestCredential
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID = %q, want %q", got.ID, orig.ID)
	}
	if got.Username != orig.Username {
		t.Errorf("Username = %q, want %q", got.Username, orig.Username)
	}
	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
	if got.Algorithm != orig.Algorithm {
		t.Errorf("Algorithm = %q, want %q", got.Algorithm, orig.Algorithm)
	}
	if got.ExpiryDate == nil || !got.ExpiryDate.Equal(expiry) {
		t.Errorf("ExpiryDate = %v, want %v", got.ExpiryDate, expiry)
	}
}

func TestIngestCredential_JSONFieldNames(t *testing.T) {
	cred := IngestCredential{ID: "c1", Username: "u1", Algorithm: HashAlgorithmMD5}
	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, want := range []string{"id", "username", "algorithm"} {
		if _, ok := m[want]; !ok {
			t.Errorf("JSON key %q missing from marshalled IngestCredential", want)
		}
	}
}

func TestIngestCredential_OptionalFieldsOmitted(t *testing.T) {
	cred := IngestCredential{ID: "c1", Username: "u1", Algorithm: HashAlgorithmSHA512}
	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"description", "expiry_date"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil", absent)
		}
	}
}

// ---- IngestCredentialCreateRequest ----

func TestIngestCredentialCreateRequest_JSONRoundTrip(t *testing.T) {
	desc := "my cred"
	req := IngestCredentialCreateRequest{
		Username:    "admin",
		Password:    "s3cr3t",
		Algorithm:   HashAlgorithmSHA512256,
		Description: &desc,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got IngestCredentialCreateRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Username != req.Username {
		t.Errorf("Username = %q, want %q", got.Username, req.Username)
	}
	if got.Password != req.Password {
		t.Errorf("Password = %q, want %q", got.Password, req.Password)
	}
	if got.Algorithm != req.Algorithm {
		t.Errorf("Algorithm = %q, want %q", got.Algorithm, req.Algorithm)
	}
	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
}

func TestIngestCredentialCreateRequest_OmitsOptionalDescription(t *testing.T) {
	req := IngestCredentialCreateRequest{
		Username:  "admin",
		Password:  "pass",
		Algorithm: HashAlgorithmMD5,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["description"]; ok {
		t.Error("description should be omitted when nil")
	}
}

// ---- IngestCredentialUpdateRequest ----

func TestIngestCredentialUpdateRequest_JSONRoundTrip(t *testing.T) {
	desc := "updated"
	expiry := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	req := IngestCredentialUpdateRequest{
		Description: &desc,
		ExpiryDate:  &expiry,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got IngestCredentialUpdateRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
	if got.ExpiryDate == nil || !got.ExpiryDate.Equal(expiry) {
		t.Errorf("ExpiryDate = %v, want %v", got.ExpiryDate, expiry)
	}
}

func TestIngestCredentialUpdateRequest_OmitsEmptyFields(t *testing.T) {
	req := IngestCredentialUpdateRequest{}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"description", "expiry_date"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil", absent)
		}
	}
}
