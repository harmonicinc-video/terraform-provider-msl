package event

import (
	"context"
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- mapEventToState ----

func TestMapEventToState_BasicFields(t *testing.T) {
	now := time.Date(2024, 3, 10, 8, 0, 0, 0, time.UTC)
	start := time.Date(2024, 3, 10, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 10, 11, 0, 0, 0, time.UTC)
	trueVal := true

	event := &models.Event{
		EventID:            "e-abc",
		EventName:          "my-event",
		StreamID:           "s-1",
		Active:             true,
		Ended:              false,
		StartTime:          &start,
		EndTime:            &end,
		SubsegmentClipping: &trueVal,
		EgressFilepaths:    []string{"/path/manifest.m3u8"},
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	var state eventResourceModel
	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}

	if state.ID.ValueString() != "e-abc" {
		t.Errorf("ID = %q, want %q", state.ID.ValueString(), "e-abc")
	}
	if state.EventName.ValueString() != "my-event" {
		t.Errorf("EventName = %q, want %q", state.EventName.ValueString(), "my-event")
	}
	if state.StreamID.ValueString() != "s-1" {
		t.Errorf("StreamID = %q, want %q", state.StreamID.ValueString(), "s-1")
	}
	if !state.Active.ValueBool() {
		t.Error("Active should be true")
	}
	if state.Ended.ValueBool() {
		t.Error("Ended should be false")
	}
	if !state.SubsegmentClipping.ValueBool() {
		t.Error("SubsegmentClipping should be true")
	}
	if state.StartTime.ValueString() != start.Format(time.RFC3339) {
		t.Errorf("StartTime = %q, want %q", state.StartTime.ValueString(), start.Format(time.RFC3339))
	}
	if state.EndTime.ValueString() != end.Format(time.RFC3339) {
		t.Errorf("EndTime = %q, want %q", state.EndTime.ValueString(), end.Format(time.RFC3339))
	}
	if state.CreatedAt.ValueString() != now.Format(time.RFC3339) {
		t.Errorf("CreatedAt = %q, want %q", state.CreatedAt.ValueString(), now.Format(time.RFC3339))
	}
}

func TestMapEventToState_NilOptionalTimes(t *testing.T) {
	event := &models.Event{
		EventID:   "e-1",
		EventName: "no-times",
		StreamID:  "s-1",
	}

	// Pre-set state fields as Unknown to check the nil-path branch.
	state := eventResourceModel{
		StartTime: types.StringUnknown(),
		EndTime:   types.StringUnknown(),
	}

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if !state.StartTime.IsNull() {
		t.Errorf("StartTime should be null when event.StartTime is nil and state was unknown, got %q", state.StartTime.ValueString())
	}
	if !state.EndTime.IsNull() {
		t.Errorf("EndTime should be null when event.EndTime is nil and state was unknown, got %q", state.EndTime.ValueString())
	}
}

// TestMapEventToState_NilStartTimeSetsNull verifies that when the API returns no
// start_time (nil), the state is set to null regardless of any prior value.
func TestMapEventToState_NilStartTimeSetsNull(t *testing.T) {
	existingStart := "2026-05-15T18:26:26Z"
	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		// StartTime intentionally nil — API did not return it
	}

	state := eventResourceModel{
		StartTime: types.StringValue(existingStart),
	}

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if !state.StartTime.IsNull() {
		t.Errorf("StartTime = %q, want null when API returns nil start_time", state.StartTime.ValueString())
	}
}

func TestMapEventToState_EgressFilepaths(t *testing.T) {
	event := &models.Event{
		EventID:         "e-1",
		EventName:       "ev",
		StreamID:        "s-1",
		EgressFilepaths: []string{"/a/b.m3u8", "/c/d.m3u8"},
	}
	var state eventResourceModel
	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if len(state.EgressFilepaths.Elements()) != 2 {
		t.Errorf("EgressFilepaths len = %d, want 2", len(state.EgressFilepaths.Elements()))
	}
}

func TestMapEventToState_EmptyEgressFilepaths(t *testing.T) {
	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
	}
	var state eventResourceModel
	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if len(state.EgressFilepaths.Elements()) != 0 {
		t.Errorf("EgressFilepaths should be empty, got %d", len(state.EgressFilepaths.Elements()))
	}
}

func TestMapEventToState_ZeroTimesNotSet(t *testing.T) {
	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		// CreatedAt and UpdatedAt are zero values
	}
	var state eventResourceModel
	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if state.CreatedAt.ValueString() != "" {
		t.Errorf("CreatedAt should be empty for zero time, got %q", state.CreatedAt.ValueString())
	}
	if state.UpdatedAt.ValueString() != "" {
		t.Errorf("UpdatedAt should be empty for zero time, got %q", state.UpdatedAt.ValueString())
	}
}

// ---- Time range validation (mirrors ValidateConfig logic) ----

func TestTimeRangeValidation_EndAfterStart_Valid(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	if !end.After(start) {
		t.Error("end should be after start")
	}
}

func TestTimeRangeValidation_EndBeforeStart_Invalid(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if end.After(start) {
		t.Error("end should NOT be after start when it is earlier")
	}
}

func TestTimeRangeValidation_EqualTimes_Invalid(t *testing.T) {
	ts := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	if ts.After(ts) {
		t.Error("equal times should not satisfy After()")
	}
}

// ---- event_name non-empty validator ----

func runEventNameValidator(t *testing.T, val types.String) bool {
	t.Helper()
	var resp validator.StringResponse
	stringvalidator.LengthAtLeast(1).ValidateString(
		context.Background(),
		validator.StringRequest{ConfigValue: val},
		&resp,
	)
	return resp.Diagnostics.HasError()
}

func TestEventNameValidator_NonEmpty_Valid(t *testing.T) {
	if runEventNameValidator(t, types.StringValue("my-event")) {
		t.Error("non-empty event_name should pass validation")
	}
}

func TestEventNameValidator_SingleChar_Valid(t *testing.T) {
	if runEventNameValidator(t, types.StringValue("x")) {
		t.Error("single-character event_name should pass validation")
	}
}

func TestEventNameValidator_Empty_Invalid(t *testing.T) {
	if !runEventNameValidator(t, types.StringValue("")) {
		t.Error("empty event_name should fail validation")
	}
}

func TestEventNameValidator_Null_Skipped(t *testing.T) {
	// Null values are skipped by LengthAtLeast — Required attribute prevents
	// null in practice, but the validator must not panic on it.
	if runEventNameValidator(t, types.StringNull()) {
		t.Error("null event_name should be skipped (not flagged) by the length validator")
	}
}

// ---- schema validation ----

// TestEventSchema_UpdatedAt_HasUseStateForUnknown verifies that updated_at
// retains UseStateForUnknown() so that no-op plans don't show a spurious diff.
// (Events have no Update path, so there is no save/restore concern here.)
func TestEventSchema_UpdatedAt_HasUseStateForUnknown(t *testing.T) {
	r := &eventResource{}
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
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("updated_at should have UseStateForUnknown to prevent spurious no-op plans")
	}
}

// ---- start_time / end_time always reflect API value ----

// TestMapEventToState_StartTimeUsesAPIValue verifies that when the API returns a
// start_time, it is written to state regardless of any prior state value.
func TestMapEventToState_StartTimeUsesAPIValue(t *testing.T) {
	priorStart := "2024-01-01T10:00:00Z"
	apiStart := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		StartTime: &apiStart,
	}
	state := eventResourceModel{
		StartTime: types.StringValue(priorStart),
	}

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if state.StartTime.ValueString() != apiStart.Format(time.RFC3339) {
		t.Errorf("StartTime = %q, want API value %q", state.StartTime.ValueString(), apiStart.Format(time.RFC3339))
	}
}

// TestMapEventToState_EndTimeUsesAPIValue verifies that when the API returns an
// end_time, it is written to state regardless of any prior state value.
func TestMapEventToState_EndTimeUsesAPIValue(t *testing.T) {
	priorEnd := "2024-01-01T12:00:00Z"
	apiEnd := time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)

	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		EndTime:   &apiEnd,
	}
	state := eventResourceModel{
		EndTime: types.StringValue(priorEnd),
	}

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if state.EndTime.ValueString() != apiEnd.Format(time.RFC3339) {
		t.Errorf("EndTime = %q, want API value %q", state.EndTime.ValueString(), apiEnd.Format(time.RFC3339))
	}
}

// TestMapEventToState_StartTimeCapturedOnCreate verifies that when state is null
// (first create), the API-returned start_time is captured into state.
func TestMapEventToState_StartTimeCapturedOnCreate(t *testing.T) {
	apiStart := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		StartTime: &apiStart,
	}
	// state.StartTime is null (zero value) — simulates create path
	var state eventResourceModel

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if state.StartTime.ValueString() != apiStart.Format(time.RFC3339) {
		t.Errorf("StartTime = %q, want API value %q", state.StartTime.ValueString(), apiStart.Format(time.RFC3339))
	}
}

// TestMapEventToState_EndTimeCapturedOnCreate verifies that end_time is captured
// from the API when state is null (first create / first server assignment).
func TestMapEventToState_EndTimeCapturedOnCreate(t *testing.T) {
	apiEnd := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	event := &models.Event{
		EventID:   "e-1",
		EventName: "ev",
		StreamID:  "s-1",
		EndTime:   &apiEnd,
	}
	var state eventResourceModel

	diags := mapEventToState(context.Background(), event, &state)
	if diags.HasError() {
		t.Fatalf("mapEventToState() returned errors: %v", diags)
	}
	if state.EndTime.ValueString() != apiEnd.Format(time.RFC3339) {
		t.Errorf("EndTime = %q, want API value %q", state.EndTime.ValueString(), apiEnd.Format(time.RFC3339))
	}
}

// ---- schema: immutable fields ----

func TestEventSchema_StreamID_IsImmutable(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["stream_id"]
	if !ok {
		t.Fatal("stream_id not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("stream_id is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("stream_id should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("stream_id should have ImmutableAfterCreation plan modifier")
	}
}

func TestEventSchema_EventName_IsImmutable(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["event_name"]
	if !ok {
		t.Fatal("event_name not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("event_name is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("event_name should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("event_name should have ImmutableAfterCreation plan modifier")
	}
}

func TestEventSchema_SourceEventName_IsImmutable(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["source_event_name"]
	if !ok {
		t.Fatal("source_event_name not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("source_event_name is not a schema.StringAttribute")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("source_event_name should have ImmutableAfterCreation plan modifier")
	}
}

func TestEventSchema_StartTime_IsImmutableAndComputed(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["start_time"]
	if !ok {
		t.Fatal("start_time not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("start_time is not a schema.StringAttribute")
	}
	if !strAttr.Computed {
		t.Error("start_time should be Computed (server may assign value)")
	}
	// Expects UseStateForUnknown + ImmutableAfterCreation = 2 modifiers
	if len(strAttr.PlanModifiers) < 2 {
		t.Errorf("start_time should have at least 2 plan modifiers, got %d", len(strAttr.PlanModifiers))
	}
}

func TestEventSchema_EndTime_IsImmutableAndComputed(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["end_time"]
	if !ok {
		t.Fatal("end_time not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("end_time is not a schema.StringAttribute")
	}
	if !strAttr.Computed {
		t.Error("end_time should be Computed (server may assign value)")
	}
	if len(strAttr.PlanModifiers) < 2 {
		t.Errorf("end_time should have at least 2 plan modifiers, got %d", len(strAttr.PlanModifiers))
	}
}

func TestEventSchema_SubsegmentClipping_IsImmutable(t *testing.T) {
	r := &eventResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["subsegment_clipping"]
	if !ok {
		t.Fatal("subsegment_clipping not found in schema")
	}
	boolAttr, ok := rawAttr.(schema.BoolAttribute)
	if !ok {
		t.Fatal("subsegment_clipping is not a schema.BoolAttribute")
	}
	if len(boolAttr.PlanModifiers) == 0 {
		t.Error("subsegment_clipping should have ImmutableAfterCreation plan modifier")
	}
}
