package stream

import (
	"context"
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// helpers

func boolPtr(b bool) *bool       { return &b }
func int64Ptr(i int64) *int64    { return &i }
func stringPtr(s string) *string { return &s }

// ---- mapStreamToState ----

func TestMapStreamToState_BasicFields(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	stream := &models.Stream{
		ID:                   "s-123",
		Description:          "test stream",
		Format:               models.StreamFormatHLS,
		OriginID:             "o-456",
		IngestLocation:       "US_SEA",
		ContractID:           "c-1",
		CPTag:                "tag1",
		GroupID:              "g-1",
		Status:               models.StreamStatusReady,
		HostName:             "ingest.example.com",
		OriginHostName:       "origin.example.com",
		PrimaryPublishingURL: "https://pub.example.com/stream",
		BackupPublishingURL:  "https://backup.example.com/stream",
		CreatedBy:            "user@example.com",
		CreatedAt:            now,
		UpdatedAt:            now,
		Archiving:            &models.Archiving{NoArchive: true},
	}

	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}

	if state.ID.ValueString() != "s-123" {
		t.Errorf("ID = %q, want %q", state.ID.ValueString(), "s-123")
	}
	if state.Description.ValueString() != "test stream" {
		t.Errorf("Description = %q, want %q", state.Description.ValueString(), "test stream")
	}
	if state.Format.ValueString() != "HLS" {
		t.Errorf("Format = %q, want %q", state.Format.ValueString(), "HLS")
	}
	if state.OriginID.ValueString() != "o-456" {
		t.Errorf("OriginID = %q, want %q", state.OriginID.ValueString(), "o-456")
	}
	if state.Status.ValueString() != "READY" {
		t.Errorf("Status = %q, want %q", state.Status.ValueString(), "READY")
	}
	if state.HostName.ValueString() != "ingest.example.com" {
		t.Errorf("HostName = %q, want %q", state.HostName.ValueString(), "ingest.example.com")
	}
	if state.PrimaryPublishingURL.ValueString() != "https://pub.example.com/stream" {
		t.Errorf("PrimaryPublishingURL = %q, want %q", state.PrimaryPublishingURL.ValueString(), "https://pub.example.com/stream")
	}
	if state.CreatedBy.ValueString() != "user@example.com" {
		t.Errorf("CreatedBy = %q, want %q", state.CreatedBy.ValueString(), "user@example.com")
	}
	if state.CreatedAt.ValueString() != now.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %q, want %q", state.CreatedAt.ValueString(), now.Format(time.RFC3339))
	}
	if state.UpdatedAt.ValueString() != now.Format(time.RFC3339) {
		t.Errorf("UpdatedAt = %q, want %q", state.UpdatedAt.ValueString(), now.Format(time.RFC3339))
	}
}

func TestMapStreamToState_ZeroTimesNotWritten(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.CreatedAt.ValueString() != "" {
		t.Errorf("CreatedAt should be empty for zero time, got %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != "" {
		t.Errorf("UpdatedAt should be empty for zero time, got %q", state.UpdatedAt.ValueString())
	}
}

func TestMapStreamToState_BackupFields_WhenPresent(t *testing.T) {
	stream := &models.Stream{
		ID:                   "s-1",
		BackupIngestLocation: stringPtr("EU_AMS"),
		BackupHostName:       stringPtr("backup.example.com"),
		BackupOriginHostName: stringPtr("backup-origin.example.com"),
		Archiving:            &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.BackupIngestLocation.ValueString() != "EU_AMS" {
		t.Errorf("BackupIngestLocation = %q, want %q", state.BackupIngestLocation.ValueString(), "EU_AMS")
	}
	if state.BackupHostName.ValueString() != "backup.example.com" {
		t.Errorf("BackupHostName = %q, want %q", state.BackupHostName.ValueString(), "backup.example.com")
	}
	if state.BackupOriginHostName.ValueString() != "backup-origin.example.com" {
		t.Errorf("BackupOriginHostName = %q, want %q", state.BackupOriginHostName.ValueString(), "backup-origin.example.com")
	}
}

func TestMapStreamToState_BackupFields_NullWhenAbsent(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if !state.BackupHostName.IsNull() {
		t.Errorf("BackupHostName should be null, got %q", state.BackupHostName.ValueString())
	}
	if !state.BackupOriginHostName.IsNull() {
		t.Errorf("BackupOriginHostName should be null, got %q", state.BackupOriginHostName.ValueString())
	}
}

func TestMapStreamToState_PlaylistDuration_FromAPI(t *testing.T) {
	stream := &models.Stream{
		ID:                    "s-1",
		PlaylistDurationInMin: int64Ptr(60),
		Archiving:             &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.PlaylistDurationInMin.ValueInt64() != 60 {
		t.Errorf("PlaylistDurationInMin = %d, want 60", state.PlaylistDurationInMin.ValueInt64())
	}
}

func TestMapStreamToState_PlaylistDuration_DefaultsToMinusOne(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.PlaylistDurationInMin.ValueInt64() != -1 {
		t.Errorf("PlaylistDurationInMin = %d, want -1", state.PlaylistDurationInMin.ValueInt64())
	}
}

func TestMapStreamToState_AllowedIPs_Populated(t *testing.T) {
	stream := &models.Stream{
		ID:         "s-1",
		AllowedIPs: []string{"192.168.1.0/24", "10.0.0.1"},
		Archiving:  &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.AllowedIPs.Elements()) != 2 {
		t.Errorf("AllowedIPs len = %d, want 2", len(state.AllowedIPs.Elements()))
	}
}

func TestMapStreamToState_AllowedIPs_EmptyList(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.AllowedIPs.Elements()) != 0 {
		t.Errorf("AllowedIPs should be empty, got %d elements", len(state.AllowedIPs.Elements()))
	}
}

// ---- ingest authentication mapping ----

func TestMapStreamToState_IngestAuth_ModeHeader(t *testing.T) {
	mode := models.IngestAuthModeHeader
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != "HEADER" {
		t.Errorf("IngestAuthenticationMode = %q, want HEADER", state.IngestAuthenticationMode.ValueString())
	}
}

func TestMapStreamToState_IngestAuth_NoneWritesNone(t *testing.T) {
	// When API returns mode=NONE, ingest_authentication_mode should be set to "NONE".
	mode := models.IngestAuthModeNone
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	state := streamResourceModel{
		IngestAuthenticationMode: types.StringNull(),
	}
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != "NONE" {
		t.Errorf("IngestAuthenticationMode = %q, want NONE", state.IngestAuthenticationMode.ValueString())
	}
}

func TestMapStreamToState_IngestAuth_NoneOverwritesHeader(t *testing.T) {
	// Regression: when the previous state had HEADER and the API now returns NONE,
	// the state must be updated to NONE rather than left as HEADER.
	mode := models.IngestAuthModeNone
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	state := streamResourceModel{
		IngestAuthenticationMode: types.StringValue("HEADER"),
	}
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != "NONE" {
		t.Errorf("IngestAuthenticationMode = %q, want NONE", state.IngestAuthenticationMode.ValueString())
	}
}

// TestMapStreamToState_IngestAuth_ModeSetsState verifies that whatever mode the
// API returns is written directly to ingest_authentication_mode in state.
func TestMapStreamToState_IngestAuth_ModeSetsState(t *testing.T) {
	mode := models.IngestAuthModeHeader
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != "HEADER" {
		t.Errorf("IngestAuthenticationMode = %q, want HEADER", state.IngestAuthenticationMode.ValueString())
	}
}

// TestMapStreamToState_IngestAuth_APIDigestSetsMode verifies that when the API
// returns mode="DIGEST", ingest_authentication_mode is set to "DIGEST" in state.
func TestMapStreamToState_IngestAuth_APIDigestSetsMode(t *testing.T) {
	mode := models.IngestAuthModeDigest
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != string(models.IngestAuthModeDigest) {
		t.Errorf("IngestAuthenticationMode = %q, want DIGEST", state.IngestAuthenticationMode.ValueString())
	}
}

// TestMapStreamToState_IngestAuth_APIDigestPreservesModeRepresentation verifies
// that when the API returns mode="DIGEST" and the plan already used
// ingest_authentication_mode="DIGEST", the mode-string representation is kept.
func TestMapStreamToState_IngestAuth_APIDigestPreservesModeRepresentation(t *testing.T) {
	mode := models.IngestAuthModeDigest
	stream := &models.Stream{
		ID:                       "s-1",
		IngestAuthenticationMode: &mode,
		Archiving:                &models.Archiving{NoArchive: true},
	}
	// Plan used ingest_authentication_mode="DIGEST".
	state := streamResourceModel{
		IngestAuthenticationMode: types.StringValue(string(models.IngestAuthModeDigest)),
	}
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if state.IngestAuthenticationMode.ValueString() != string(models.IngestAuthModeDigest) {
		t.Errorf("IngestAuthenticationMode = %q, want %q", state.IngestAuthenticationMode.ValueString(), string(models.IngestAuthModeDigest))
	}
}

// ---- archiving block ----

func TestMapStreamToState_Archiving_NoArchive(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.Archiving.Elements()) != 1 {
		t.Fatalf("Archiving should have 1 element, got %d", len(state.Archiving.Elements()))
	}
	var archivings []archivingModel
	diags = state.Archiving.ElementsAs(context.Background(), &archivings, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(archiving): %v", diags)
	}
	if !archivings[0].NoArchive.ValueBool() {
		t.Error("NoArchive should be true")
	}
	if len(archivings[0].AutomaticPurge.Elements()) != 0 {
		t.Error("AutomaticPurge should be empty when NoArchive=true")
	}
}

func TestMapStreamToState_Archiving_WithPurge(t *testing.T) {
	stream := &models.Stream{
		ID: "s-1",
		Archiving: &models.Archiving{
			NoArchive:      false,
			AutomaticPurge: &models.AutomaticPurge{RetentionDays: int64Ptr(30)},
		},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	var archivings []archivingModel
	diags = state.Archiving.ElementsAs(context.Background(), &archivings, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(archiving): %v", diags)
	}
	var purges []automaticPurgeModel
	diags = archivings[0].AutomaticPurge.ElementsAs(context.Background(), &purges, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(automatic_purge): %v", diags)
	}
	if len(purges) != 1 {
		t.Fatalf("AutomaticPurge should have 1 element, got %d", len(purges))
	}
	if purges[0].RetentionDays.ValueInt64() != 30 {
		t.Errorf("RetentionDays = %d, want 30", purges[0].RetentionDays.ValueInt64())
	}
}

// ---- ingest_header block ----

func TestMapStreamToState_IngestHeader_Populated(t *testing.T) {
	stream := &models.Stream{
		ID: "s-1",
		IngestHeader: &models.StreamIngestHeader{
			Header: stringPtr("X-Auth-Token"),
			Values: []string{"token1", "token2"},
		},
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.IngestHeader.Elements()) != 1 {
		t.Fatalf("IngestHeader should have 1 element, got %d", len(state.IngestHeader.Elements()))
	}
	var headers []ingestHeaderModel
	diags = state.IngestHeader.ElementsAs(context.Background(), &headers, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(ingest_header): %v", diags)
	}
	if headers[0].Header.ValueString() != "X-Auth-Token" {
		t.Errorf("Header = %q, want %q", headers[0].Header.ValueString(), "X-Auth-Token")
	}
	if len(headers[0].Values.Elements()) != 2 {
		t.Errorf("Values len = %d, want 2", len(headers[0].Values.Elements()))
	}
}

func TestMapStreamToState_IngestHeader_Nil(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.IngestHeader.Elements()) != 0 {
		t.Errorf("IngestHeader should be empty, got %d elements", len(state.IngestHeader.Elements()))
	}
}

// ---- playback block ----

func TestMapStreamToState_Playback_WithG2O(t *testing.T) {
	stream := &models.Stream{
		ID: "s-1",
		Playback: &models.Playback{
			AkamaiG2oAuth: &models.AkamaiG2oAuth{
				Enabled:    boolPtr(true),
				G2oVersion: int64Ptr(4),
				SecretKey:  stringPtr("mysecret"),
				TimeDelta:  int64Ptr(30),
			},
		},
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.Playback.Elements()) != 1 {
		t.Fatalf("Playback should have 1 element, got %d", len(state.Playback.Elements()))
	}
	var pbs []playbackModel
	diags = state.Playback.ElementsAs(context.Background(), &pbs, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(playback): %v", diags)
	}
	var g2os []akamaiG2oAuthModel
	diags = pbs[0].AkamaiG2oAuth.ElementsAs(context.Background(), &g2os, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(akamai_g2o_auth): %v", diags)
	}
	if len(g2os) != 1 {
		t.Fatalf("AkamaiG2oAuth should have 1 element, got %d", len(g2os))
	}
	if !g2os[0].Enabled.ValueBool() {
		t.Error("G2O Enabled should be true")
	}
	if g2os[0].G2oVersion.ValueInt64() != 4 {
		t.Errorf("G2oVersion = %d, want 4", g2os[0].G2oVersion.ValueInt64())
	}
	if g2os[0].TimeDelta.ValueInt64() != 30 {
		t.Errorf("TimeDelta = %d, want 30", g2os[0].TimeDelta.ValueInt64())
	}
}

func TestMapStreamToState_Playback_SecretKeyPreservedFromState(t *testing.T) {
	// The API never returns secret_key; the existing state value must be preserved.
	existingKey, _ := types.ListValue(
		types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes},
		[]attr.Value{},
	)
	g2oObj, _ := types.ObjectValue(akamaiG2oAuthAttrTypes, map[string]attr.Value{
		"enabled":     types.BoolValue(true),
		"g2o_version": types.Int64Value(4),
		"secret_key":  types.StringValue("preserved-secret"),
		"time_delta":  types.Int64Value(30),
	})
	g2oList, _ := types.ListValue(types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes}, []attr.Value{g2oObj})
	pbObj, _ := types.ObjectValue(playbackAttrTypes, map[string]attr.Value{
		"akamai_g2o_auth": g2oList,
	})
	pbList, _ := types.ListValue(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{pbObj})
	_ = existingKey

	stream := &models.Stream{
		ID: "s-1",
		Playback: &models.Playback{
			AkamaiG2oAuth: &models.AkamaiG2oAuth{
				Enabled:    boolPtr(true),
				G2oVersion: int64Ptr(4),
				// SecretKey intentionally omitted — API never returns it
				TimeDelta: int64Ptr(30),
			},
		},
		Archiving: &models.Archiving{NoArchive: true},
	}
	state := streamResourceModel{Playback: pbList}
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	var pbs []playbackModel
	diags = state.Playback.ElementsAs(context.Background(), &pbs, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(playback): %v", diags)
	}
	var g2os []akamaiG2oAuthModel
	diags = pbs[0].AkamaiG2oAuth.ElementsAs(context.Background(), &g2os, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(akamai_g2o_auth): %v", diags)
	}
	if g2os[0].SecretKey.ValueString() != "preserved-secret" {
		t.Errorf("SecretKey = %q, want %q", g2os[0].SecretKey.ValueString(), "preserved-secret")
	}
}

func TestMapStreamToState_Playback_Nil(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.Playback.Elements()) != 0 {
		t.Errorf("Playback should be empty, got %d elements", len(state.Playback.Elements()))
	}
}

// ---- hls_to_llhls block ----

func TestMapStreamToState_HlsToLlHls_Populated(t *testing.T) {
	stream := &models.Stream{
		ID: "s-1",
		HlsToLlHls: &models.HlsToLlHlsConfig{
			Enabled:         true,
			SegmentTemplate: `segment-(\d+)\.ts`,
		},
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.HlsToLlHls.Elements()) != 1 {
		t.Fatalf("HlsToLlHls should have 1 element, got %d", len(state.HlsToLlHls.Elements()))
	}
	var cfgs []hlsToLlHlsModel
	diags = state.HlsToLlHls.ElementsAs(context.Background(), &cfgs, false)
	if diags.HasError() {
		t.Fatalf("ElementsAs(hls_to_llhls): %v", diags)
	}
	if !cfgs[0].Enabled.ValueBool() {
		t.Error("HlsToLlHls Enabled should be true")
	}
	if cfgs[0].SegmentTemplate.ValueString() != `segment-(\d+)\.ts` {
		t.Errorf("SegmentTemplate = %q, want %q", cfgs[0].SegmentTemplate.ValueString(), `segment-(\d+)\.ts`)
	}
}

func TestMapStreamToState_HlsToLlHls_Nil(t *testing.T) {
	stream := &models.Stream{
		ID:        "s-1",
		Archiving: &models.Archiving{NoArchive: true},
	}
	var state streamResourceModel
	diags := mapStreamToState(context.Background(), stream, &state)
	if diags.HasError() {
		t.Fatalf("mapStreamToState() returned errors: %v", diags)
	}
	if len(state.HlsToLlHls.Elements()) != 0 {
		t.Errorf("HlsToLlHls should be empty, got %d elements", len(state.HlsToLlHls.Elements()))
	}
}

// ---- archivingFromState ----

func TestArchivingFromState_NoArchive(t *testing.T) {
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(true),
		"automatic_purge": purgeList,
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	result, diags := archivingFromState(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("archivingFromState() returned errors: %v", diags)
	}
	if !result.NoArchive {
		t.Error("NoArchive should be true")
	}
	if result.AutomaticPurge != nil {
		t.Error("AutomaticPurge should be nil when NoArchive=true")
	}
}

func TestArchivingFromState_WithPurge(t *testing.T) {
	purgeObj, _ := types.ObjectValue(automaticPurgeAttrTypes, map[string]attr.Value{
		"retention_days": types.Int64Value(14),
	})
	purgeList, _ := types.ListValue(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{purgeObj})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(false),
		"automatic_purge": purgeList,
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	result, diags := archivingFromState(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("archivingFromState() returned errors: %v", diags)
	}
	if result.NoArchive {
		t.Error("NoArchive should be false")
	}
	if result.AutomaticPurge == nil {
		t.Fatal("AutomaticPurge should not be nil")
	}
	if *result.AutomaticPurge.RetentionDays != 14 {
		t.Errorf("RetentionDays = %d, want 14", *result.AutomaticPurge.RetentionDays)
	}
}

// ---- ingestHeaderFromState ----

func TestIngestHeaderFromState_Nil(t *testing.T) {
	result, diags := ingestHeaderFromState(context.Background(), types.ListNull(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}))
	if diags.HasError() {
		t.Fatalf("ingestHeaderFromState() returned errors: %v", diags)
	}
	if result != nil {
		t.Error("result should be nil for null list")
	}
}

func TestIngestHeaderFromState_Populated(t *testing.T) {
	valsAttr, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("v1"),
		types.StringValue("v2"),
	})
	obj, _ := types.ObjectValue(ingestHeaderAttrTypes, map[string]attr.Value{
		"header": types.StringValue("X-Custom"),
		"values": valsAttr,
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{obj})

	result, diags := ingestHeaderFromState(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("ingestHeaderFromState() returned errors: %v", diags)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if *result.Header != "X-Custom" {
		t.Errorf("Header = %q, want %q", *result.Header, "X-Custom")
	}
	if len(result.Values) != 2 {
		t.Errorf("Values len = %d, want 2", len(result.Values))
	}
}

// ---- playbackFromState ----

func TestPlaybackFromState_Nil(t *testing.T) {
	result, diags := playbackFromState(context.Background(), types.ListNull(types.ObjectType{AttrTypes: playbackAttrTypes}))
	if diags.HasError() {
		t.Fatalf("playbackFromState() returned errors: %v", diags)
	}
	if result != nil {
		t.Error("result should be nil for null list")
	}
}

func TestPlaybackFromState_WithG2O(t *testing.T) {
	g2oObj, _ := types.ObjectValue(akamaiG2oAuthAttrTypes, map[string]attr.Value{
		"enabled":     types.BoolValue(true),
		"g2o_version": types.Int64Value(4),
		"secret_key":  types.StringValue("secret"),
		"time_delta":  types.Int64Value(15),
	})
	g2oList, _ := types.ListValue(types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes}, []attr.Value{g2oObj})
	pbObj, _ := types.ObjectValue(playbackAttrTypes, map[string]attr.Value{
		"akamai_g2o_auth": g2oList,
	})
	pbList, _ := types.ListValue(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{pbObj})

	result, diags := playbackFromState(context.Background(), pbList)
	if diags.HasError() {
		t.Fatalf("playbackFromState() returned errors: %v", diags)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.AkamaiG2oAuth == nil {
		t.Fatal("AkamaiG2oAuth should not be nil")
	}
	if !*result.AkamaiG2oAuth.Enabled {
		t.Error("Enabled should be true")
	}
	if *result.AkamaiG2oAuth.G2oVersion != 4 {
		t.Errorf("G2oVersion = %d, want 4", *result.AkamaiG2oAuth.G2oVersion)
	}
	if *result.AkamaiG2oAuth.SecretKey != "secret" {
		t.Errorf("SecretKey = %q, want %q", *result.AkamaiG2oAuth.SecretKey, "secret")
	}
	if *result.AkamaiG2oAuth.TimeDelta != 15 {
		t.Errorf("TimeDelta = %d, want 15", *result.AkamaiG2oAuth.TimeDelta)
	}
}

// ---- hlsToLlHlsFromState ----

func TestHlsToLlHlsFromState_Nil(t *testing.T) {
	result, diags := hlsToLlHlsFromState(context.Background(), types.ListNull(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}))
	if diags.HasError() {
		t.Fatalf("hlsToLlHlsFromState() returned errors: %v", diags)
	}
	if result != nil {
		t.Error("result should be nil for null list")
	}
}

func TestHlsToLlHlsFromState_Populated(t *testing.T) {
	obj, _ := types.ObjectValue(hlsToLlHlsAttrTypes, map[string]attr.Value{
		"enabled":          types.BoolValue(true),
		"segment_template": types.StringValue(`seg-(\d+)`),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{obj})

	result, diags := hlsToLlHlsFromState(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("hlsToLlHlsFromState() returned errors: %v", diags)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if !result.Enabled {
		t.Error("Enabled should be true")
	}
	if result.SegmentTemplate != `seg-(\d+)` {
		t.Errorf("SegmentTemplate = %q, want %q", result.SegmentTemplate, `seg-(\d+)`)
	}
}

// ---- buildCreateRequest ----

func TestBuildCreateRequest_BasicFields(t *testing.T) {
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(true),
		"automatic_purge": purgeList,
	})
	archList, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	plan := streamResourceModel{
		Description:    types.StringValue("my stream"),
		Format:         types.StringValue("HLS"),
		OriginID:       types.StringValue("o-1"),
		IngestLocation: types.StringValue("US_SEA"),
		ContractID:     types.StringValue("c-1"),
		CPTag:          types.StringValue("tag1"),
		GroupID:        types.StringValue("g-1"),
		Archiving:      archList,
		IngestHeader:   types.ListValueMust(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{}),
		Playback:       types.ListValueMust(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{}),
		HlsToLlHls:     types.ListValueMust(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{}),
	}

	req, diags := buildCreateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("buildCreateRequest() returned errors: %v", diags)
	}
	if req.Description != "my stream" {
		t.Errorf("Description = %q, want %q", req.Description, "my stream")
	}
	if string(req.Format) != "HLS" {
		t.Errorf("Format = %q, want HLS", req.Format)
	}
	if req.OriginID != "o-1" {
		t.Errorf("OriginID = %q, want %q", req.OriginID, "o-1")
	}
	if req.Archiving.NoArchive != true {
		t.Error("Archiving.NoArchive should be true")
	}
}

func TestBuildCreateRequest_OptionalFields(t *testing.T) {
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(true),
		"automatic_purge": purgeList,
	})
	archList, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	plan := streamResourceModel{
		Format:                types.StringValue("CMAF"),
		OriginID:              types.StringValue("o-1"),
		IngestLocation:        types.StringValue("US_SEA"),
		BackupIngestLocation:  types.StringValue("EU_AMS"),
		ContractID:            types.StringValue("c-1"),
		CPTag:                 types.StringValue("tag1"),
		GroupID:               types.StringValue("g-1"),
		PlaylistDurationInMin: types.Int64Value(60),
		AllowedIPs: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("10.0.0.0/8"),
		}),
		IngestAuthenticationMode: types.StringValue(string(models.IngestAuthModeDigest)),
		Archiving:                archList,
		IngestHeader:             types.ListValueMust(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{}),
		Playback:                 types.ListValueMust(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{}),
		HlsToLlHls:               types.ListValueMust(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{}),
	}

	req, diags := buildCreateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("buildCreateRequest() returned errors: %v", diags)
	}
	if req.BackupIngestLocation == nil || *req.BackupIngestLocation != "EU_AMS" {
		t.Errorf("BackupIngestLocation = %v, want EU_AMS", req.BackupIngestLocation)
	}
	if req.PlaylistDurationInMin == nil || *req.PlaylistDurationInMin != 60 {
		t.Errorf("PlaylistDurationInMin = %v, want 60", req.PlaylistDurationInMin)
	}
	if len(req.AllowedIPs) != 1 || req.AllowedIPs[0] != "10.0.0.0/8" {
		t.Errorf("AllowedIPs = %v, want [10.0.0.0/8]", req.AllowedIPs)
	}
	if req.IngestAuthenticationMode == nil || *req.IngestAuthenticationMode != models.IngestAuthModeDigest {
		t.Errorf("IngestAuthenticationMode = %v, want DIGEST", req.IngestAuthenticationMode)
	}
}

// ---- buildUpdateRequest ----

func TestBuildUpdateRequest_BasicFields(t *testing.T) {
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(false),
		"automatic_purge": purgeList,
	})
	archList, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	plan := streamResourceModel{
		Description:  types.StringValue("updated"),
		GroupID:      types.StringValue("g-new"),
		Archiving:    archList,
		IngestHeader: types.ListValueMust(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{}),
		Playback:     types.ListValueMust(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{}),
		HlsToLlHls:   types.ListValueMust(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{}),
	}

	req, diags := buildUpdateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("buildUpdateRequest() returned errors: %v", diags)
	}
	if req.Description == nil || *req.Description != "updated" {
		t.Errorf("Description = %v, want updated", req.Description)
	}
	if req.GroupID == nil || *req.GroupID != "g-new" {
		t.Errorf("GroupID = %v, want g-new", req.GroupID)
	}
	if req.Archiving == nil {
		t.Error("Archiving should not be nil in update request")
	}
	// AllowedIPs is always sent (empty = unrestricted)
	if req.AllowedIPs == nil {
		t.Error("AllowedIPs should never be nil in update request")
	}
}

func TestBuildUpdateRequest_AuthMode(t *testing.T) {
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	archObj, _ := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(true),
		"automatic_purge": purgeList,
	})
	archList, _ := types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})

	plan := streamResourceModel{
		IngestAuthenticationMode: types.StringValue("DIGEST"),
		Archiving:                archList,
		IngestHeader:             types.ListValueMust(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{}),
		Playback:                 types.ListValueMust(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{}),
		HlsToLlHls:               types.ListValueMust(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{}),
	}

	req, diags := buildUpdateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("buildUpdateRequest() returned errors: %v", diags)
	}
	if req.IngestAuthenticationMode == nil || string(*req.IngestAuthenticationMode) != "DIGEST" {
		t.Errorf("IngestAuthenticationMode = %v, want DIGEST", req.IngestAuthenticationMode)
	}
}

// ---- schema validation ----

func TestStreamSchema_AllowedIPs_IsRequired(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["allowed_ips"]
	if !ok {
		t.Fatal("allowed_ips not found in schema")
	}
	listAttr, ok := rawAttr.(schema.ListAttribute)
	if !ok {
		t.Fatal("allowed_ips is not a schema.ListAttribute")
	}
	if !listAttr.Required {
		t.Error("allowed_ips should be Required")
	}
	if listAttr.Optional {
		t.Error("allowed_ips should not be Optional")
	}
	if listAttr.Computed {
		t.Error("allowed_ips should not be Computed")
	}
}

func TestStreamSchema_AllowedIPs_HasTwoValidators(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["allowed_ips"]
	if !ok {
		t.Fatal("allowed_ips not found in schema")
	}
	listAttr, ok := rawAttr.(schema.ListAttribute)
	if !ok {
		t.Fatal("allowed_ips is not a schema.ListAttribute")
	}
	// Expect SizeAtMost(270) + ValueStringsAre(cidrStringValidator{})
	if len(listAttr.Validators) != 2 {
		t.Errorf("allowed_ips should have 2 validators (SizeAtMost + ValueStringsAre), got %d", len(listAttr.Validators))
	}
}

func TestStreamSchema_Format_HasOneOfValidator(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["format"]
	if !ok {
		t.Fatal("format not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("format is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("format should be Required")
	}
	if len(strAttr.Validators) == 0 {
		t.Error("format should have at least one validator (OneOf)")
	}
	// Verify ImmutableAfterCreation plan modifier is present
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("format should have ImmutableAfterCreation plan modifier")
	}
}

func TestStreamSchema_OriginID_IsImmutable(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["origin_id"]
	if !ok {
		t.Fatal("origin_id not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("origin_id is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("origin_id should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("origin_id should have ImmutableAfterCreation plan modifier")
	}
}

func TestStreamSchema_IngestLocation_IsImmutable(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["ingest_location"]
	if !ok {
		t.Fatal("ingest_location not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("ingest_location is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("ingest_location should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("ingest_location should have ImmutableAfterCreation plan modifier")
	}
}

func TestStreamSchema_BackupIngestLocation_IsImmutableAndComputed(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["backup_ingest_location"]
	if !ok {
		t.Fatal("backup_ingest_location not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("backup_ingest_location is not a schema.StringAttribute")
	}
	if !strAttr.Optional {
		t.Error("backup_ingest_location should be Optional")
	}
	if !strAttr.Computed {
		t.Error("backup_ingest_location should be Computed")
	}
	// Expects UseStateForUnknown + ImmutableAfterCreation = 2 modifiers
	if len(strAttr.PlanModifiers) < 2 {
		t.Errorf("backup_ingest_location should have at least 2 plan modifiers (UseStateForUnknown + ImmutableAfterCreation), got %d", len(strAttr.PlanModifiers))
	}
}

func TestStreamSchema_ContractID_IsImmutable(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["contract_id"]
	if !ok {
		t.Fatal("contract_id not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("contract_id is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("contract_id should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("contract_id should have ImmutableAfterCreation plan modifier")
	}
}

func TestStreamSchema_CPTag_IsImmutable(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["cptag"]
	if !ok {
		t.Fatal("cptag not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("cptag is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("cptag should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("cptag should have ImmutableAfterCreation plan modifier")
	}
}

func TestStreamSchema_PlaylistDuration_HasValidator(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["playlist_duration_in_min"]
	if !ok {
		t.Fatal("playlist_duration_in_min not found in schema")
	}
	intAttr, ok := rawAttr.(schema.Int64Attribute)
	if !ok {
		t.Fatal("playlist_duration_in_min is not a schema.Int64Attribute")
	}
	if len(intAttr.Validators) == 0 {
		t.Error("playlist_duration_in_min should have at least one validator (Any(-1, Between(0,720)))")
	}
}

func TestStreamSchema_PlaylistDuration_ValidatorBehavior(t *testing.T) {
	r := &streamResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	intAttr := schemaResp.Schema.Attributes["playlist_duration_in_min"].(schema.Int64Attribute)

	validValues := []int64{-1, 0, 1, 360, 720}
	invalidValues := []int64{-2, -100, 721, 1000}

	for _, v := range validValues {
		val := types.Int64Value(v)
		req := validator.Int64Request{ConfigValue: val}
		var diagResp validator.Int64Response
		for _, vd := range intAttr.Validators {
			vd.ValidateInt64(context.Background(), req, &diagResp)
		}
		if diagResp.Diagnostics.HasError() {
			t.Errorf("expected %d to be valid, but got errors: %s", v, diagResp.Diagnostics)
		}
	}

	for _, v := range invalidValues {
		val := types.Int64Value(v)
		req := validator.Int64Request{ConfigValue: val}
		var diagResp validator.Int64Response
		for _, vd := range intAttr.Validators {
			vd.ValidateInt64(context.Background(), req, &diagResp)
		}
		if !diagResp.Diagnostics.HasError() {
			t.Errorf("expected %d to be invalid, but validator passed", v)
		}
	}
}

func TestStreamSchema_UpdatedAt_NoUseStateForUnknown(t *testing.T) {
	r := &streamResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["updated_at"]
	if !ok {
		t.Fatal("updated_at not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("updated_at is not a schema.StringAttribute")
	}
	if len(strAttr.PlanModifiers) != 0 {
		t.Errorf("updated_at must have no plan modifiers (got %d); UseStateForUnknown causes spurious drift after updates", len(strAttr.PlanModifiers))
	}
}
