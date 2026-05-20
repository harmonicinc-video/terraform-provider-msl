package models

import "time"

// StreamFormat represents the streaming format enum.
type StreamFormat string

const (
	// StreamFormatHLS represents HLS streaming format.
	StreamFormatHLS StreamFormat = "HLS"
	// StreamFormatCMAF represents CMAF streaming format.
	StreamFormatCMAF StreamFormat = "CMAF"
	// StreamFormatDASH represents DASH streaming format.
	StreamFormatDASH StreamFormat = "DASH"
)

// StreamStatus represents the stream lifecycle status.
type StreamStatus string

const (
	// StreamStatusCreating indicates the stream is being created.
	StreamStatusCreating StreamStatus = "CREATING"
	// StreamStatusReady indicates the stream is ready for use.
	StreamStatusReady StreamStatus = "READY"
	// StreamStatusDeleted indicates the stream has been deleted.
	StreamStatusDeleted StreamStatus = "DELETED"
)

// IngestAuthMode represents ingest authentication modes.
type IngestAuthMode string

const (
	// IngestAuthModeNone disables ingest authentication.
	IngestAuthModeNone IngestAuthMode = "NONE"
	// IngestAuthModeDigest enables HTTP Digest authentication.
	IngestAuthModeDigest IngestAuthMode = "DIGEST"
	// IngestAuthModeHeader enables header-based authentication.
	IngestAuthModeHeader IngestAuthMode = "HEADER"
)

// AkamaiG2oAuth holds Akamai G2O playback authentication settings.
type AkamaiG2oAuth struct {
	Enabled    *bool   `json:"enabled,omitempty"`
	G2oVersion *int64  `json:"g2o_version,omitempty"`
	SecretKey  *string `json:"secret_key,omitempty"`
	TimeDelta  *int64  `json:"time_delta,omitempty"`
}

// Playback holds playback configuration for a stream.
type Playback struct {
	AkamaiG2oAuth *AkamaiG2oAuth `json:"akamai_g2o_auth,omitempty"`
}

// AutomaticPurge holds archiving purge settings.
type AutomaticPurge struct {
	RetentionDays *int64 `json:"retention_days,omitempty"`
}

// Archiving holds archiving configuration for a stream.
type Archiving struct {
	NoArchive      bool            `json:"no_archive"`
	AutomaticPurge *AutomaticPurge `json:"automatic_purge,omitempty"`
}

// StreamIngestHeader holds ingest header authentication configuration.
type StreamIngestHeader struct {
	Header *string  `json:"header,omitempty"`
	Values []string `json:"values,omitempty"`
}

// HlsToLlHlsConfig holds HLS-to-LL-HLS conversion settings.
type HlsToLlHlsConfig struct {
	Enabled         bool   `json:"enabled"`
	SegmentTemplate string `json:"segment_template"`
}

// Stream represents an MSL5 Stream resource as returned by the API.
type Stream struct {
	ID                       string              `json:"stream_id"`
	Description              string              `json:"description,omitempty"`
	Format                   StreamFormat        `json:"format"`
	Status                   StreamStatus        `json:"status,omitempty"`
	OriginID                 string              `json:"origin_id"`
	IngestLocation           string              `json:"ingest_location"`
	BackupIngestLocation     *string             `json:"backup_ingest_location,omitempty"`
	HostName                 string              `json:"host_name,omitempty"`
	BackupHostName           *string             `json:"backup_host_name,omitempty"`
	OriginHostName           string              `json:"origin_host_name,omitempty"`
	BackupOriginHostName     *string             `json:"backup_origin_host_name,omitempty"`
	PrimaryPublishingURL     string              `json:"primary_publishing_url,omitempty"`
	BackupPublishingURL      string              `json:"backup_publishing_url,omitempty"`
	PlaylistDurationInMin    *int64              `json:"playlist_duration_in_min,omitempty"`
	AllowedIPs               []string            `json:"allowed_ips,omitempty"`
	IngestAuthentication     *bool               `json:"ingest_authentication,omitempty"`
	IngestAuthenticationMode *IngestAuthMode     `json:"ingest_authentication_mode,omitempty"`
	IngestHeader             *StreamIngestHeader `json:"ingest_header,omitempty"`
	Playback                 *Playback           `json:"playback,omitempty"`
	Archiving                *Archiving          `json:"archiving,omitempty"`
	HlsToLlHls               *HlsToLlHlsConfig   `json:"hls_to_llhls,omitempty"`
	ContractID               string              `json:"contract_id"`
	CPTag                    string              `json:"cptag"`
	GroupID                  string              `json:"group_id"`
	CreatedBy                string              `json:"created_by,omitempty"`
	CreatedAt                time.Time           `json:"created_at,omitempty"`
	UpdatedAt                time.Time           `json:"updated_at,omitempty"`
}

// StreamCreateRequest is the payload for POST /api/v1/streams.
type StreamCreateRequest struct {
	Description              string              `json:"description"`
	Format                   StreamFormat        `json:"format"`
	OriginID                 string              `json:"origin_id"`
	IngestLocation           string              `json:"ingest_location"`
	BackupIngestLocation     *string             `json:"backup_ingest_location,omitempty"`
	ContractID               string              `json:"contract_id"`
	CPTag                    string              `json:"cptag"`
	GroupID                  string              `json:"group_id"`
	Archiving                Archiving           `json:"archiving"`
	Playback                 *Playback           `json:"playback,omitempty"`
	PlaylistDurationInMin    *int64              `json:"playlist_duration_in_min,omitempty"`
	AllowedIPs               []string            `json:"allowed_ips,omitempty"`
	IngestAuthentication     *bool               `json:"ingest_authentication,omitempty"`
	IngestAuthenticationMode *IngestAuthMode     `json:"ingest_authentication_mode,omitempty"`
	IngestHeader             *StreamIngestHeader `json:"ingest_header,omitempty"`
	HlsToLlHls               *HlsToLlHlsConfig   `json:"hls_to_llhls,omitempty"`
}

// StreamUpdateRequest is the payload for PUT /api/v1/streams/{id}.
// The API uses full-replace semantics: omitted optional fields reset to defaults.
type StreamUpdateRequest struct {
	Description              *string             `json:"description,omitempty"`
	GroupID                  *string             `json:"group_id,omitempty"`
	Archiving                *Archiving          `json:"archiving,omitempty"`
	Playback                 *Playback           `json:"playback,omitempty"`
	PlaylistDurationInMin    *int64              `json:"playlist_duration_in_min,omitempty"`
	AllowedIPs               []string            `json:"allowed_ips"`
	IngestAuthentication     *bool               `json:"ingest_authentication,omitempty"`
	IngestAuthenticationMode *IngestAuthMode     `json:"ingest_authentication_mode,omitempty"`
	IngestHeader             *StreamIngestHeader `json:"ingest_header,omitempty"`
	HlsToLlHls               *HlsToLlHlsConfig   `json:"hls_to_llhls,omitempty"`
}

// ListStreamsResponse wraps the top-level list response.
type ListStreamsResponse struct {
	Streams []Stream `json:"streams"`
}
