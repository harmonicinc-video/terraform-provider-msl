package planmodifiers_test

import (
	"context"
	"testing"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// stateWithID builds a minimal tfsdk.State that contains only an "id" attribute
// set to the given value. It is used to simulate an already-created resource in
// plan modifier tests.
func stateWithID(id string) tfsdk.State {
	s := schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
		},
	}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}},
		map[string]tftypes.Value{"id": tftypes.NewValue(tftypes.String, id)},
	)
	return tfsdk.State{Schema: s, Raw: raw}
}

func TestImmutableAfterCreation_NoErrorOnCreate(t *testing.T) {
	// State is null (resource being created) — any value must be accepted.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("US_SEA"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error on create, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_NoErrorWhenUnchanged(t *testing.T) {
	// State and plan have the same value — no error.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("US_SEA"),
		PlanValue:  types.StringValue("US_SEA"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when value unchanged, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_ErrorWhenValueChanges(t *testing.T) {
	// State has a value and plan differs — must error.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("US_SEA"),
		PlanValue:  types.StringValue("US_WEST"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected error when immutable value changes, got none")
	}
}

func TestImmutableAfterCreation_NoErrorWhenPlanUnknown(t *testing.T) {
	// Plan value is unknown (unresolved reference) — must not error.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("US_SEA"),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown plan value, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_NoErrorWhenStateUnknown(t *testing.T) {
	// State is unknown — treated as creation, must not error.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringUnknown(),
		PlanValue:  types.StringValue("US_SEA"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when state is unknown, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_ErrorWhenFieldRemoved(t *testing.T) {
	// State has a value; plan is null (field removed from config after creation).
	// Applies to optional immutable fields like backup_ingest_location.
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringValue("US_SEA"),
		PlanValue:  types.StringNull(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected error when immutable field is removed after creation, got none")
	}
}

// ---- PlanModifyBool ----

func TestImmutableAfterCreation_Bool_NoErrorOnCreate(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.BoolRequest{
		StateValue: types.BoolNull(),
		PlanValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	mod.PlanModifyBool(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error on create, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_Bool_NoErrorWhenUnchanged(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(true),
		PlanValue:  types.BoolValue(true),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	mod.PlanModifyBool(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error when value unchanged, got: %v", resp.Diagnostics)
	}
}

func TestImmutableAfterCreation_Bool_ErrorWhenValueChanges(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(true),
		PlanValue:  types.BoolValue(false),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	mod.PlanModifyBool(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected error when immutable bool value changes, got none")
	}
}

func TestImmutableAfterCreation_Bool_NoErrorWhenPlanUnknown(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.BoolRequest{
		StateValue: types.BoolValue(true),
		PlanValue:  types.BoolUnknown(),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	mod.PlanModifyBool(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown plan value, got: %v", resp.Diagnostics)
	}
}

// TestImmutableAfterCreation_SuppressDriftOnExistingResource covers the regression
// where a user adds start_time / end_time to an already-created event's config.
// The resource exists (has an ID in state) but the attribute was never set at
// creation time (null in state). The modifier must lock the plan value to null so
// that no spurious update is scheduled.
func TestImmutableAfterCreation_SuppressDriftOnExistingResource(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("2024-01-01T00:00:00Z"),
		State:      stateWithID("de8fac0e-6750-475d-a0ee-1a80982748d7"),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	mod.PlanModifyString(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %v", resp.Diagnostics)
	}
	if !resp.PlanValue.IsNull() {
		t.Errorf("expected plan value to be locked to null (suppress drift), got: %s", resp.PlanValue)
	}
}

func TestImmutableAfterCreation_Bool_SuppressDriftOnExistingResource(t *testing.T) {
	mod := planmodifiers.ImmutableAfterCreation{}
	req := planmodifier.BoolRequest{
		StateValue: types.BoolNull(),
		PlanValue:  types.BoolValue(true),
		State:      stateWithID("de8fac0e-6750-475d-a0ee-1a80982748d7"),
	}
	resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
	mod.PlanModifyBool(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error, got: %v", resp.Diagnostics)
	}
	if !resp.PlanValue.IsNull() {
		t.Errorf("expected plan value to be locked to null (suppress drift), got: %v", resp.PlanValue)
	}
}
