package stream

import (
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func streamAttrString(obj types.Object, key string) string {
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

func streamAttrInt64(obj types.Object, key string) int64 {
	v, ok := obj.Attributes()[key]
	if !ok {
		return 0
	}
	iv, ok := v.(types.Int64)
	if !ok {
		return 0
	}
	return iv.ValueInt64()
}

// ---- streamSummaryToObject ----

func TestStreamSummaryToObject_BasicFields(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	s := models.Stream{
		ID:                   "s-1",
		Description:          "my stream",
		Format:               models.StreamFormatHLS,
		Status:               models.StreamStatusReady,
		OriginID:             "o-1",
		IngestLocation:       "US_SEA",
		ContractID:           "c-1",
		CPTag:                "tag1",
		GroupID:              "g-1",
		HostName:             "ingest.example.com",
		PrimaryPublishingURL: "https://pub.example.com",
		BackupPublishingURL:  "https://backup.example.com",
		CreatedBy:            "user@example.com",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}

	cases := map[string]string{
		"id":                     "s-1",
		"description":            "my stream",
		"format":                 "HLS",
		"status":                 "READY",
		"origin_id":              "o-1",
		"ingest_location":        "US_SEA",
		"contract_id":            "c-1",
		"cptag":                  "tag1",
		"group_id":               "g-1",
		"host_name":              "ingest.example.com",
		"primary_publishing_url": "https://pub.example.com",
		"backup_publishing_url":  "https://backup.example.com",
		"created_by":             "user@example.com",
		"created_at":             now.Format(time.RFC3339),
		"updated_at":             now.Format(time.RFC3339),
	}
	for key, want := range cases {
		if got := streamAttrString(obj, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestStreamSummaryToObject_BackupIngestLocation_Present(t *testing.T) {
	loc := "EU_AMS"
	s := models.Stream{
		ID:                   "s-1",
		BackupIngestLocation: &loc,
	}
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	if got := streamAttrString(obj, "backup_ingest_location"); got != "EU_AMS" {
		t.Errorf("backup_ingest_location = %q, want EU_AMS", got)
	}
}

func TestStreamSummaryToObject_BackupIngestLocation_Nil(t *testing.T) {
	s := models.Stream{ID: "s-1"} // no backup location
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	if got := streamAttrString(obj, "backup_ingest_location"); got != "" {
		t.Errorf("backup_ingest_location should be empty string for nil, got %q", got)
	}
}

func TestStreamSummaryToObject_PlaylistDuration_Present(t *testing.T) {
	dur := int64(60)
	s := models.Stream{
		ID:                    "s-1",
		PlaylistDurationInMin: &dur,
	}
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	if got := streamAttrInt64(obj, "playlist_duration_in_min"); got != 60 {
		t.Errorf("playlist_duration_in_min = %d, want 60", got)
	}
}

func TestStreamSummaryToObject_PlaylistDuration_NilDefaultsToMinusOne(t *testing.T) {
	s := models.Stream{ID: "s-1"} // no playlist duration
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	if got := streamAttrInt64(obj, "playlist_duration_in_min"); got != -1 {
		t.Errorf("playlist_duration_in_min = %d, want -1", got)
	}
}

func TestStreamSummaryToObject_ZeroTimesProduceEmptyStrings(t *testing.T) {
	s := models.Stream{ID: "s-1"} // zero CreatedAt / UpdatedAt
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	if got := streamAttrString(obj, "created_at"); got != "" {
		t.Errorf("created_at should be empty for zero time, got %q", got)
	}
	if got := streamAttrString(obj, "updated_at"); got != "" {
		t.Errorf("updated_at should be empty for zero time, got %q", got)
	}
}

func TestStreamSummaryToObject_ContainsAllExpectedKeys(t *testing.T) {
	s := models.Stream{ID: "s-1"}
	obj, diags := streamSummaryToObject(s)
	if diags.HasError() {
		t.Fatalf("streamSummaryToObject() returned errors: %v", diags)
	}
	for key := range streamAttrTypes {
		if _, ok := obj.Attributes()[key]; !ok {
			t.Errorf("missing attribute %q in object", key)
		}
	}
}
