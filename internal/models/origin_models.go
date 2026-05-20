// Package models defines the shared data models for the Terraform provider.
package models

import "time"

// OriginStatus represents the origin lifecycle status.
type OriginStatus string

const (
	// OriginStatusCreating indicates the origin is being provisioned.
	OriginStatusCreating OriginStatus = "CREATING"
	// OriginStatusReady indicates the origin is ready for use.
	OriginStatusReady OriginStatus = "READY"
	// OriginStatusDeleted indicates the origin has been deleted.
	OriginStatusDeleted OriginStatus = "DELETED"
)

// SharedKey represents a shared key entry on an Origin.
type SharedKey struct {
	Name     string `json:"name"`
	Key      string `json:"key"`
	HostName string `json:"host_name"`
	Enabled  bool   `json:"enabled"`
}

// Origin represents an MSL5 Origin resource as returned by the API.
type Origin struct {
	ID                   string       `json:"origin_id"`
	AccountID            string       `json:"account_id,omitempty"`
	HostName             string       `json:"host_name"`
	IngestLocation       string       `json:"ingest_location"`
	BackupIngestLocation string       `json:"backup_ingest_location,omitempty"`
	ContractID           string       `json:"contract_id"`
	CPTag                string       `json:"cptag"`
	GroupID              string       `json:"group_id"`
	SharedKeys           []SharedKey  `json:"shared_keys,omitempty"`
	Status               OriginStatus `json:"status,omitempty"`
	BackupHostName       string       `json:"backup_host_name,omitempty"`
	CreatedAt            time.Time    `json:"created_at,omitempty"`
	UpdatedAt            time.Time    `json:"updated_at,omitempty"`
	CreatedBy            string       `json:"created_by,omitempty"`
}

// OriginCreateRequest is the payload for POST /api/v1/origins.
type OriginCreateRequest struct {
	HostName             string      `json:"host_name"`
	IngestLocation       string      `json:"ingest_location"`
	BackupIngestLocation string      `json:"backup_ingest_location,omitempty"`
	ContractID           string      `json:"contract_id"`
	CPTag                string      `json:"cptag"`
	GroupID              string      `json:"group_id"`
	SharedKeys           []SharedKey `json:"shared_keys,omitempty"`
}

// OriginUpdateRequest is the payload for PUT /api/v1/origins/{id}.
// Only group_id and shared_keys are updatable per the API design.
type OriginUpdateRequest struct {
	GroupID    string      `json:"group_id,omitempty"`
	SharedKeys []SharedKey `json:"shared_keys,omitempty"`
}

// ListOriginsResponse wraps the top-level list response.
type ListOriginsResponse struct {
	Origins []Origin `json:"origins"`
}
