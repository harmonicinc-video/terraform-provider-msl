package models

import (
	"encoding/json"
	"testing"
	"time"
)

// ---- Origin JSON round-trip ----

func TestOrigin_JSONRoundTrip(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	orig := Origin{
		ID:                   "origin-1",
		HostName:             "host.example.com",
		IngestLocation:       "US_ORD",
		BackupIngestLocation: "GB_LON",
		ContractID:           "ctr-1",
		CPTag:                "tag1",
		GroupID:              "grp-1",
		Status:               "READY",
		CreatedAt:            now,
		UpdatedAt:            now,
		CreatedBy:            "user@example.com",
		SharedKeys: []SharedKey{
			{Name: "key1", Key: "secret", HostName: "cdn.example.com", Enabled: true},
		},
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got Origin
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID = %q, want %q", got.ID, orig.ID)
	}
	if got.HostName != orig.HostName {
		t.Errorf("HostName = %q, want %q", got.HostName, orig.HostName)
	}
	if got.BackupIngestLocation != orig.BackupIngestLocation {
		t.Errorf("BackupIngestLocation = %q, want %q", got.BackupIngestLocation, orig.BackupIngestLocation)
	}
	if len(got.SharedKeys) != 1 {
		t.Fatalf("len(SharedKeys) = %d, want 1", len(got.SharedKeys))
	}
	if got.SharedKeys[0].Key != "secret" {
		t.Errorf("SharedKeys[0].Key = %q, want %q", got.SharedKeys[0].Key, "secret")
	}
}

func TestOrigin_JSONFieldNames(t *testing.T) {
	orig := Origin{ID: "o1", HostName: "h.example.com", ContractID: "c1"}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, want := range []string{"origin_id", "host_name", "contract_id"} {
		if _, ok := m[want]; !ok {
			t.Errorf("JSON key %q missing from marshalled Origin", want)
		}
	}
}

// ---- OriginCreateRequest ----

func TestOriginCreateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := OriginCreateRequest{
		HostName:       "h.example.com",
		IngestLocation: "US_ORD",
		ContractID:     "c1",
		CPTag:          "tag1",
		GroupID:        "g1",
		// BackupIngestLocation and SharedKeys intentionally omitted
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["backup_ingest_location"]; ok {
		t.Error("backup_ingest_location should be omitted when empty")
	}
	if _, ok := m["shared_keys"]; ok {
		t.Error("shared_keys should be omitted when nil")
	}
}

// ---- OriginUpdateRequest ----

func TestOriginUpdateRequest_OmitsEmptyFields(t *testing.T) {
	req := OriginUpdateRequest{GroupID: "g2"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["group_id"]; !ok {
		t.Error("group_id should be present when set")
	}
	if _, ok := m["shared_keys"]; ok {
		t.Error("shared_keys should be omitted when nil")
	}
}

// ---- ListOriginsResponse ----

func TestListOriginsResponse_Unmarshal(t *testing.T) {
	raw := `{"origins":[{"origin_id":"o1"},{"origin_id":"o2"}]}`
	var resp ListOriginsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(resp.Origins) != 2 {
		t.Fatalf("len(Origins) = %d, want 2", len(resp.Origins))
	}
	if resp.Origins[0].ID != "o1" {
		t.Errorf("Origins[0].ID = %q, want %q", resp.Origins[0].ID, "o1")
	}
}

// ---- SharedKey ----

func TestSharedKey_JSONRoundTrip(t *testing.T) {
	sk := SharedKey{Name: "k1", Key: "secret", HostName: "cdn.example.com", Enabled: true}
	data, err := json.Marshal(sk)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	var got SharedKey
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if got != sk {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, sk)
	}
}
