// Package planmodifiers provides shared Terraform plan modifier implementations.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ImmutableAfterCreation is a plan modifier that prevents a string attribute
// from being changed after the resource has been created. Unlike RequiresReplace,
// this surfaces an explicit error instead of silently scheduling a replacement,
// ensuring that users cannot accidentally destroy a resource by editing these
// fields in their configuration.
type ImmutableAfterCreation struct{}

// Description returns a plain-text description of the plan modifier.
func (m ImmutableAfterCreation) Description(_ context.Context) string {
	return "This attribute is immutable and can only be set during resource creation."
}

// MarkdownDescription returns a Markdown description of the plan modifier.
func (m ImmutableAfterCreation) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyString enforces immutability by returning an error if the attribute is changed after resource creation.
func (m ImmutableAfterCreation) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		// State value is null/unknown. This could be a brand-new resource OR an
		// existing resource where the attribute was never set at creation time.
		// Check for the latter by looking for a non-null "id" in state: if an ID
		// exists the resource is already created, so lock the plan to the state
		// value (null) to suppress any config-driven drift.
		if resourceExistsInState(ctx, req.State) {
			if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
				// Config is now providing a value for an attribute that was null
				// at creation time. Emit an immutability error but do NOT override
				// the plan value — leaving it as the config value avoids the
				// "Provider produced invalid plan" mismatch that Terraform reports
				// when planned null != config non-null.
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Immutable Attribute",
					"This attribute is immutable and can only be set during resource creation. To change it, you must destroy and recreate the resource.",
				)
				return
			}
			resp.PlanValue = req.StateValue
		}
		return
	}
	// Plan value is not yet resolved — cannot compare; allow to proceed.
	if req.PlanValue.IsUnknown() {
		return
	}
	// Value is unchanged — no issue.
	if req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Immutable Attribute",
		"This attribute is immutable and can only be set during resource creation. To change it, you must destroy and recreate the resource.",
	)
}

// PlanModifyBool implements planmodifier.Bool with the same immutability semantics.
func (m ImmutableAfterCreation) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		if resourceExistsInState(ctx, req.State) {
			if !req.ConfigValue.IsNull() && !req.ConfigValue.IsUnknown() {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Immutable Attribute",
					"This attribute is immutable and can only be set during resource creation. To change it, you must destroy and recreate the resource.",
				)
				return
			}
			resp.PlanValue = req.StateValue
		}
		return
	}
	if req.PlanValue.IsUnknown() {
		return
	}
	if req.PlanValue.Equal(req.StateValue) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Immutable Attribute",
		"This attribute is immutable and can only be set during resource creation. To change it, you must destroy and recreate the resource.",
	)
}

// resourceExistsInState returns true when the state already contains a non-null,
// non-unknown "id" attribute, indicating that the resource was previously created.
// It is used to distinguish a genuine Create (empty state) from an Update where
// an optional attribute was simply never set at creation time.
func resourceExistsInState(ctx context.Context, state tfsdk.State) bool {
	// A zero-value State (no schema) means we are on the Create path.
	if state.Schema == nil {
		return false
	}
	var id types.String
	if diags := state.GetAttribute(ctx, path.Root("id"), &id); diags.HasError() {
		return false
	}
	return !id.IsNull() && !id.IsUnknown()
}
