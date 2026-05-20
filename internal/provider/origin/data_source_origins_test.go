package origin

import (
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attrString is a test helper that reads a string attribute from a types.Object.
func dsAttrString(obj types.Object, key string) string {
	v, ok := obj.Attributes()[key]
	if !ok {
		return ""
	}
	sv, ok := v.(types.String)
	if !ok {
		return ""
	}
	return sv.ValueString()
}

// ---- originSummaryToObject ----

func TestOriginSummaryToObject_BasicFields(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	o := models.Origin{
		ID:                   "o-123",
		AccountID:            "acct-456",
		HostName:             "host.example.com",
		IngestLocation:       "US_SEA",
		BackupIngestLocation: "EU_AMS",
		ContractID:           "c-1",
		CPTag:                "tag1",
		GroupID:              "g-1",
		Status:               "READY",
		CreatedBy:            "user@example.com",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	obj, diags := originSummaryToObject(o)
	if diags.HasError() {
		t.Fatalf("originSummaryToObject() returned errors: %v", diags)
	}

	cases := map[string]string{
		"id":                     "o-123",
		"account_id":             "acct-456",
		"host_name":              "host.example.com",
		"ingest_location":        "US_SEA",
		"backup_ingest_location": "EU_AMS",
		"contract_id":            "c-1",
		"cptag":                  "tag1",
		"group_id":               "g-1",
		"status":                 "READY",
		"created_by":             "user@example.com",
		"created_at":             now.Format(time.RFC3339),
		"updated_at":             now.Format(time.RFC3339),
	}
	for key, want := range cases {
		if got := dsAttrString(obj, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestOriginSummaryToObject_ZeroTimesProduceEmptyStrings(t *testing.T) {
	o := models.Origin{ID: "o-1"} // zero CreatedAt / UpdatedAt
	obj, diags := originSummaryToObject(o)
	if diags.HasError() {
		t.Fatalf("originSummaryToObject() returned errors: %v", diags)
	}
	if got := dsAttrString(obj, "created_at"); got != "" {
		t.Errorf("created_at should be empty for zero time, got %q", got)
	}
	if got := dsAttrString(obj, "updated_at"); got != "" {
		t.Errorf("updated_at should be empty for zero time, got %q", got)
	}
}

func TestOriginSummaryToObject_EmptyBackupIngestLocation(t *testing.T) {
	o := models.Origin{ID: "o-1"} // no BackupIngestLocation
	obj, diags := originSummaryToObject(o)
	if diags.HasError() {
		t.Fatalf("originSummaryToObject() returned errors: %v", diags)
	}
	if got := dsAttrString(obj, "backup_ingest_location"); got != "" {
		t.Errorf("backup_ingest_location should be empty string, got %q", got)
	}
}

func TestOriginSummaryToObject_ContainsAllExpectedKeys(t *testing.T) {
	o := models.Origin{ID: "o-1"}
	obj, diags := originSummaryToObject(o)
	if diags.HasError() {
		t.Fatalf("originSummaryToObject() returned errors: %v", diags)
	}
	for key := range originAttrTypes {
		if _, ok := obj.Attributes()[key]; !ok {
			t.Errorf("missing attribute %q in object", key)
		}
	}
}

func TestOriginSummaryToObject_TimestampFormat(t *testing.T) {
	ts := time.Date(2025, 1, 15, 9, 30, 0, 0, time.UTC)
	o := models.Origin{
		ID:        "o-1",
		CreatedAt: ts,
		UpdatedAt: ts,
	}
	obj, diags := originSummaryToObject(o)
	if diags.HasError() {
		t.Fatalf("originSummaryToObject() returned errors: %v", diags)
	}
	want := "2025-01-15T09:30:00Z"
	if got := dsAttrString(obj, "created_at"); got != want {
		t.Errorf("created_at = %q, want %q", got, want)
	}
	if got := dsAttrString(obj, "updated_at"); got != want {
		t.Errorf("updated_at = %q, want %q", got, want)
	}
}
