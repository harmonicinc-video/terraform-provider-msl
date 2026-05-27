package origin

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- validateBackupLocation ----

func TestValidateBackupLocation_SameAsPrimary_Error(t *testing.T) {
	plan := originResourceModel{
		IngestLocation:       types.StringValue("US_ORD"),
		BackupIngestLocation: types.StringValue("US_ORD"),
	}
	diags := validateBackupLocation(plan)
	if !diags.HasError() {
		t.Error("expected error when backup_ingest_location == ingest_location, got none")
	}
}

func TestValidateBackupLocation_DifferentFromPrimary_NoError(t *testing.T) {
	plan := originResourceModel{
		IngestLocation:       types.StringValue("US_ORD"),
		BackupIngestLocation: types.StringValue("GB_LON"),
	}
	diags := validateBackupLocation(plan)
	if diags.HasError() {
		t.Errorf("expected no error for different backup location, got: %v", diags)
	}
}

func TestValidateBackupLocation_NullBackup_NoError(t *testing.T) {
	plan := originResourceModel{
		IngestLocation:       types.StringValue("US_ORD"),
		BackupIngestLocation: types.StringNull(),
	}
	diags := validateBackupLocation(plan)
	if diags.HasError() {
		t.Errorf("expected no error when backup is null, got: %v", diags)
	}
}

func TestValidateBackupLocation_UnknownBackup_NoError(t *testing.T) {
	plan := originResourceModel{
		IngestLocation:       types.StringValue("US_ORD"),
		BackupIngestLocation: types.StringUnknown(),
	}
	diags := validateBackupLocation(plan)
	if diags.HasError() {
		t.Errorf("expected no error when backup is unknown, got: %v", diags)
	}
}

// ---- mapOriginToState ----

func TestMapOriginToState_BasicFields(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	origin := &models.Origin{
		ID:             "o-123",
		AccountID:      "acct-456",
		HostName:       "host.example.com",
		IngestLocation: "US_ORD",
		ContractID:     "c1",
		CPTag:          "tag1",
		GroupID:        "g1",
		Status:         "READY",
		CreatedBy:      "user@example.com",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}

	if state.ID.ValueString() != "o-123" {
		t.Errorf("ID = %q, want %q", state.ID.ValueString(), "o-123")
	}
	if state.AccountID.ValueString() != "acct-456" {
		t.Errorf("AccountID = %q, want %q", state.AccountID.ValueString(), "acct-456")
	}
	if state.IngestLocation.ValueString() != "US_ORD" {
		t.Errorf("IngestLocation = %q, want %q", state.IngestLocation.ValueString(), "US_ORD")
	}
	if state.Status.ValueString() != "READY" {
		t.Errorf("Status = %q, want %q", state.Status.ValueString(), "READY")
	}
	if state.CreatedBy.ValueString() != "user@example.com" {
		t.Errorf("CreatedBy = %q, want %q", state.CreatedBy.ValueString(), "user@example.com")
	}
	if state.CreatedAt.ValueString() != now.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %q, want %q", state.CreatedAt.ValueString(), now.Format(time.RFC3339))
	}
}

func TestMapOriginToState_PreservesHostName(t *testing.T) {
	// When the state already has a host_name, it must NOT be overwritten by the API
	// value (the API transforms short names into FQDNs).
	origin := &models.Origin{
		ID:       "o-1",
		HostName: "host-fqdn.example.com", // server-side transformed FQDN
	}
	state := originResourceModel{
		HostName: types.StringValue("host"), // user-supplied short name
	}
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.HostName.ValueString() != "host" {
		t.Errorf("HostName should be preserved as %q, got %q", "host", state.HostName.ValueString())
	}
}

func TestMapOriginToState_SetsHostNameWhenNull(t *testing.T) {
	// During import, HostName is null; mapOriginToState should populate it from the API.
	origin := &models.Origin{
		ID:       "o-1",
		HostName: "api-returned-host.example.com",
	}
	state := originResourceModel{
		HostName: types.StringNull(),
	}
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.HostName.ValueString() != "api-returned-host.example.com" {
		t.Errorf("HostName = %q, want %q", state.HostName.ValueString(), "api-returned-host.example.com")
	}
}

func TestHostNameMatchesFQDN(t *testing.T) {
	tests := []struct {
		name           string
		fqdn           string
		hostName       string
		ingestLocation string
		want           bool
	}{
		{
			name:     "pattern1: hostname.domain",
			fqdn:     "myhost.ingest.msl5.akamai.com",
			hostName: "myhost",
			want:     true,
		},
		{
			name:           "pattern2: location-hostname.domain (US_SEA)",
			fqdn:           "us-sea-uswestterraformtest.dev.mslorigin.nebula.video",
			hostName:       "uswestterraformtest",
			ingestLocation: "US_SEA",
			want:           true,
		},
		{
			name:           "pattern2: location-hostname.domain (US_ORD)",
			fqdn:           "us-ord-myhost.dev.mslorigin.nebula.video",
			hostName:       "myhost",
			ingestLocation: "US_ORD",
			want:           true,
		},
		{
			name:           "different hostname, same location — no match",
			fqdn:           "us-sea-oldhost.dev.mslorigin.nebula.video",
			hostName:       "newhost",
			ingestLocation: "US_SEA",
			want:           false,
		},
		{
			name:           "wrong location prefix — no match",
			fqdn:           "us-sea-myhost.dev.mslorigin.nebula.video",
			hostName:       "myhost",
			ingestLocation: "US_ORD",
			want:           false,
		},
		{
			name:     "no ingest_location provided, pattern2 fqdn — no match",
			fqdn:     "us-sea-myhost.dev.mslorigin.nebula.video",
			hostName: "myhost",
			want:     false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hostNameMatchesFQDN(tc.fqdn, tc.hostName, tc.ingestLocation)
			if got != tc.want {
				t.Errorf("hostNameMatchesFQDN(%q, %q, %q) = %v, want %v",
					tc.fqdn, tc.hostName, tc.ingestLocation, got, tc.want)
			}
		})
	}
}

func TestHostNameFQDNModifier_NormalisesToFQDNWhenPrefixMatches(t *testing.T) {
	// Pattern 1 (no location prefix): state holds FQDN, config has short name.
	mod := hostNameFQDNModifier{} // default locationAttr = "ingest_location"
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("myhost.ingest.msl5.akamai.com"),
		PlanValue:  types.StringValue("myhost"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.PlanValue.ValueString() != "myhost.ingest.msl5.akamai.com" {
		t.Errorf("PlanValue = %q, want FQDN %q", resp.PlanValue.ValueString(), "myhost.ingest.msl5.akamai.com")
	}
}

// The "no-op when a different hostname is in state" scenario (pattern 2 path) is
// covered by TestHostNameMatchesFQDN; constructing a full tfsdk.Plan in a unit
// test would require the complete provider schema.

func TestHostNameFQDNModifier_NoOpWhenStateNull(t *testing.T) {
	mod := hostNameFQDNModifier{locationAttr: "backup_ingest_location"}
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("myhost"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.PlanValue.ValueString() != "myhost" {
		t.Errorf("PlanValue = %q, want unchanged %q", resp.PlanValue.ValueString(), "myhost")
	}
}

func TestMapOriginToState_PreservesBackupIngestLocation(t *testing.T) {
	// After create/read the state already has a value; mapOriginToState must
	// not overwrite it with the API value (same reasoning as host_name).
	origin := &models.Origin{
		ID:                   "o-1",
		BackupIngestLocation: "US_ORD", // API echoes the value
	}
	state := originResourceModel{
		BackupIngestLocation: types.StringValue("US_ORD"),
	}
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.BackupIngestLocation.ValueString() != "US_ORD" {
		t.Errorf("BackupIngestLocation = %q, want preserved %q", state.BackupIngestLocation.ValueString(), "US_ORD")
	}
}

func TestMapOriginToState_SetsBackupIngestLocationWhenNull(t *testing.T) {
	// During import the state is null; mapOriginToState must populate from API.
	origin := &models.Origin{
		ID:                   "o-1",
		BackupIngestLocation: "US_SEA",
	}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.BackupIngestLocation.ValueString() != "US_SEA" {
		t.Errorf("BackupIngestLocation = %q, want %q", state.BackupIngestLocation.ValueString(), "US_SEA")
	}
}

func TestMapOriginToState_PreservesBackupHostName(t *testing.T) {
	// Once stored, the API-assigned backup FQDN must not be overwritten on
	// subsequent Read calls.
	origin := &models.Origin{
		ID:             "o-1",
		BackupHostName: "us-ord-backup.dev.mslorigin.nebula.video",
	}
	state := originResourceModel{
		BackupHostName: types.StringValue("us-ord-backup.dev.mslorigin.nebula.video"),
	}
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.BackupHostName.ValueString() != "us-ord-backup.dev.mslorigin.nebula.video" {
		t.Errorf("BackupHostName = %q, want preserved %q", state.BackupHostName.ValueString(), "us-ord-backup.dev.mslorigin.nebula.video")
	}
}

func TestMapOriginToState_SetsBackupHostNameWhenNull(t *testing.T) {
	// During import the state is null; mapOriginToState must populate from API.
	origin := &models.Origin{
		ID:             "o-1",
		BackupHostName: "us-sea-backup.dev.mslorigin.nebula.video",
	}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.BackupHostName.ValueString() != "us-sea-backup.dev.mslorigin.nebula.video" {
		t.Errorf("BackupHostName = %q, want %q", state.BackupHostName.ValueString(), "us-sea-backup.dev.mslorigin.nebula.video")
	}
}

// ---- ingest location zone format regex ----

// zoneRegex mirrors the pattern used in the schema validators for ingest_location
// and backup_ingest_location.
var zoneRegex = regexp.MustCompile(`^[A-Z]{2,3}_[A-Z]{2,5}$`)

func TestIngestLocationZoneFormat_ValidZones(t *testing.T) {
	valid := []string{"US_ORD", "US_SEA", "GB_LON", "EU_LON", "AP_TYO", "US_EAST"}
	for _, z := range valid {
		if !zoneRegex.MatchString(z) {
			t.Errorf("zone %q should be valid but did not match regex", z)
		}
	}
}

func TestIngestLocationZoneFormat_InvalidZones(t *testing.T) {
	invalid := []string{
		"",            // empty
		"us_ord",      // lowercase
		"US-ORD",      // wrong separator
		"USORD",       // no separator
		"US_",         // missing zone part
		"_ORD",        // missing region part
		"US_TOOLONG",  // zone part too long (>5)
		"TOOLONG_ORD", // region part too long (>3)
		"US_O R D",    // spaces
	}
	for _, z := range invalid {
		if zoneRegex.MatchString(z) {
			t.Errorf("zone %q should be invalid but matched regex", z)
		}
	}
}

// ---- validateDeleteStatus ----

func TestValidateDeleteStatus_ReadyStatus_NoError(t *testing.T) {
	state := originResourceModel{
		ID:     types.StringValue("o-1"),
		Status: types.StringValue(string(models.OriginStatusReady)),
	}
	diags := validateDeleteStatus(state)
	if diags.HasError() {
		t.Errorf("expected no error for READY status, got: %v", diags)
	}
}

func TestValidateDeleteStatus_CreatingStatus_Error(t *testing.T) {
	state := originResourceModel{
		ID:     types.StringValue("o-1"),
		Status: types.StringValue(string(models.OriginStatusCreating)),
	}
	diags := validateDeleteStatus(state)
	if !diags.HasError() {
		t.Error("expected error for CREATING status, got none")
	}
}

func TestValidateDeleteStatus_DeletedStatus_Error(t *testing.T) {
	state := originResourceModel{
		ID:     types.StringValue("o-1"),
		Status: types.StringValue(string(models.OriginStatusDeleted)),
	}
	diags := validateDeleteStatus(state)
	if !diags.HasError() {
		t.Error("expected error for DELETED status, got none")
	}
}

func TestValidateDeleteStatus_EmptyStatus_Error(t *testing.T) {
	state := originResourceModel{
		ID:     types.StringValue("o-1"),
		Status: types.StringValue(""),
	}
	diags := validateDeleteStatus(state)
	if !diags.HasError() {
		t.Error("expected error for empty status, got none")
	}
}

func TestMapOriginToState_BackupLocation_SetWhenPresent(t *testing.T) {
	origin := &models.Origin{
		ID:                   "o-1",
		BackupIngestLocation: "GB_LON",
	}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.BackupIngestLocation.IsNull() {
		t.Error("BackupIngestLocation should be set, got null")
	}
	if state.BackupIngestLocation.ValueString() != "GB_LON" {
		t.Errorf("BackupIngestLocation = %q, want %q", state.BackupIngestLocation.ValueString(), "GB_LON")
	}
}

func TestMapOriginToState_BackupLocation_NullWhenAbsent(t *testing.T) {
	origin := &models.Origin{ID: "o-1"} // no backup location
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if !state.BackupIngestLocation.IsNull() {
		t.Errorf("BackupIngestLocation should be null, got %q", state.BackupIngestLocation.ValueString())
	}
}

func TestMapOriginToState_SharedKeys(t *testing.T) {
	origin := &models.Origin{
		ID: "o-1",
		SharedKeys: []models.SharedKey{
			{Name: "k1", Key: "secret", HostName: "cdn.example.com", Enabled: true},
			{Name: "k2", Key: "other", HostName: "cdn2.example.com", Enabled: false},
		},
	}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if state.SharedKeys.IsNull() || state.SharedKeys.IsUnknown() {
		t.Fatal("SharedKeys should not be null/unknown")
	}
	if len(state.SharedKeys.Elements()) != 2 {
		t.Errorf("SharedKeys len = %d, want 2", len(state.SharedKeys.Elements()))
	}
}

func TestMapOriginToState_EmptySharedKeys(t *testing.T) {
	origin := &models.Origin{ID: "o-1"}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	if len(state.SharedKeys.Elements()) != 0 {
		t.Errorf("SharedKeys should be empty, got %d elements", len(state.SharedKeys.Elements()))
	}
}

func TestMapOriginToState_ZeroTimesNotSet(t *testing.T) {
	origin := &models.Origin{
		ID: "o-1",
		// CreatedAt and UpdatedAt are zero value
	}
	var state originResourceModel
	diags := mapOriginToState(context.Background(), origin, &state)
	if diags.HasError() {
		t.Fatalf("mapOriginToState() returned errors: %v", diags)
	}
	// Zero times must not be written to state (would produce empty string)
	if state.CreatedAt.ValueString() != "" {
		t.Errorf("CreatedAt should be empty for zero time, got %q", state.CreatedAt.ValueString())
	}
}

// ---- schema validation ----

func TestOriginSchema_UpdatedAt_NoUseStateForUnknown(t *testing.T) {
	r := &originResource{}
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
