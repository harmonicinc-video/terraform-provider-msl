package models

import "time"

// HashAlgorithm represents the password hashing algorithm for ingest credentials.
type HashAlgorithm string

const (
	// HashAlgorithmSHA256 uses SHA-256 for password hashing.
	HashAlgorithmSHA256 HashAlgorithm = "SHA256"
	// HashAlgorithmSHA512256 uses SHA-512/256 for password hashing.
	HashAlgorithmSHA512256 HashAlgorithm = "SHA512_256"
	// HashAlgorithmSHA512 uses SHA-512 for password hashing.
	HashAlgorithmSHA512 HashAlgorithm = "SHA512"
	// HashAlgorithmMD5 uses MD5 for password hashing.
	HashAlgorithmMD5 HashAlgorithm = "MD5"
)

// IngestCredential represents an MSL5 Ingest Credential resource as returned by the API.
// The password is never returned by the API (server-side hashed).
type IngestCredential struct {
	ID          string        `json:"id"`
	Username    string        `json:"username"`
	Description *string       `json:"description,omitempty"`
	Algorithm   HashAlgorithm `json:"algorithm"`
	ExpiryDate  *time.Time    `json:"expiry_date,omitempty"`
}

// IngestCredentialCreateRequest is the payload for POST /api/v1/streams/{stream_id}/ingest_credentials.
type IngestCredentialCreateRequest struct {
	Username    string        `json:"username"`
	Password    string        `json:"password"`
	Algorithm   HashAlgorithm `json:"algorithm"`
	Description *string       `json:"description,omitempty"`
}

// IngestCredentialUpdateRequest is the payload for PUT /api/v1/streams/{stream_id}/ingest_credentials/{id}.
// Only description and expiry_date are updatable.
type IngestCredentialUpdateRequest struct {
	Description *string    `json:"description,omitempty"`
	ExpiryDate  *time.Time `json:"expiry_date,omitempty"`
}
