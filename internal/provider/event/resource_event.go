// Package event contains the Terraform resource and data source implementations for MSL events.
package event

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure eventResource implements required interfaces.
var _ resource.Resource = &eventResource{}
var _ resource.ResourceWithImportState = &eventResource{}
var _ resource.ResourceWithValidateConfig = &eventResource{}

// NewEventResource is the factory function registered with the provider.
func NewEventResource() resource.Resource {
	return &eventResource{}
}

type eventResource struct {
	client *client.Client
}

// --- State model ---

type eventResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	StreamID           types.String `tfsdk:"stream_id"`
	EventName          types.String `tfsdk:"event_name"`
	SourceEventName    types.String `tfsdk:"source_event_name"`
	StartTime          types.String `tfsdk:"start_time"`
	EndTime            types.String `tfsdk:"end_time"`
	SubsegmentClipping types.Bool   `tfsdk:"subsegment_clipping"`
	// Computed
	Active          types.Bool   `tfsdk:"active"`
	Ended           types.Bool   `tfsdk:"ended"`
	EgressFilepaths types.Set    `tfsdk:"egress_filepaths"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

// --- Metadata / Schema ---

func (r *eventResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event"
}

func (r *eventResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an MSL5 **Event** resource (`/api/v1/streams/{stream_id}/events`). Events are immutable after creation — all configuration attributes can only be set at creation time.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique event ID assigned by the MSL5 API (`event_id`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stream_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the stream this event belongs to. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"event_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique name of the event within the stream. Must be non-empty. Used as the path identifier in API calls. Immutable: can only be set during resource creation.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"source_event_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Source event name for past-event clipping. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"start_time": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "RFC3339 start time for the event. When omitted, the API assigns a default. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"end_time": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "RFC3339 end time for the event. Must be after `start_time`. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"subsegment_clipping": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable sub-segment clipping (`#EXT-X-START:TIME-OFFSET`). Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.Bool{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			// Computed
			"active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the event is currently active.",
			},
			"ended": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the event has ended.",
			},
			"egress_filepaths": schema.SetAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Set of multivariant manifest paths that have been ingested for this event.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of when the event was created.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of the last update.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// --- Validation ---

func (r *eventResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config eventResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.StartTime.IsNull() || config.StartTime.IsUnknown() ||
		config.EndTime.IsNull() || config.EndTime.IsUnknown() {
		return
	}

	start, err := time.Parse(time.RFC3339, config.StartTime.ValueString())
	if err != nil {
		return // format errors are caught during apply
	}
	end, err := time.Parse(time.RFC3339, config.EndTime.ValueString())
	if err != nil {
		return
	}

	if !end.After(start) {
		resp.Diagnostics.AddAttributeError(
			path.Root("end_time"),
			"Invalid time range",
			"end_time must be after start_time.",
		)
	}
}

// --- Lifecycle ---

func (r *eventResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *eventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := models.EventCreateRequest{
		EventName: plan.EventName.ValueString(),
	}

	if !plan.SourceEventName.IsNull() && !plan.SourceEventName.IsUnknown() {
		v := plan.SourceEventName.ValueString()
		apiReq.SourceEventName = &v
	}
	if !plan.StartTime.IsNull() && !plan.StartTime.IsUnknown() {
		t, err := time.Parse(time.RFC3339, plan.StartTime.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("start_time"),
				"Invalid start_time format",
				fmt.Sprintf("start_time must be RFC3339: %s", err),
			)
			return
		}
		apiReq.StartTime = &t
	}
	if !plan.EndTime.IsNull() && !plan.EndTime.IsUnknown() {
		t, err := time.Parse(time.RFC3339, plan.EndTime.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("end_time"),
				"Invalid end_time format",
				fmt.Sprintf("end_time must be RFC3339: %s", err),
			)
			return
		}
		apiReq.EndTime = &t
	}
	if !plan.SubsegmentClipping.IsNull() && !plan.SubsegmentClipping.IsUnknown() {
		v := plan.SubsegmentClipping.ValueBool()
		apiReq.SubsegmentClipping = &v
	}

	event, err := r.client.CreateEvent(ctx, plan.StreamID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating event", err.Error())
		return
	}

	resp.Diagnostics.Append(mapEventToState(ctx, event, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *eventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state eventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	event, err := r.client.GetEvent(ctx, state.StreamID.ValueString(), state.EventName.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading event", err.Error())
		return
	}

	resp.Diagnostics.Append(mapEventToState(ctx, event, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is not implemented — events are immutable after creation.
func (r *eventResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Events cannot be updated",
		"The MSL5 API has no update operation for events. All configuration attributes are immutable after creation.",
	)
}

func (r *eventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch current status from the API — local state may be stale.
	current, err := r.client.GetEvent(ctx, state.StreamID.ValueString(), state.EventName.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return // already gone
		}
		resp.Diagnostics.AddError("Error reading event before delete", err.Error())
		return
	}
	if current.Active {
		resp.Diagnostics.AddError(
			"Cannot delete an active event",
			fmt.Sprintf("Event %q in stream %q is currently active. Wait for it to end before destroying it.", state.EventName.ValueString(), state.StreamID.ValueString()),
		)
		return
	}

	if err := r.client.DeleteEvent(ctx, state.StreamID.ValueString(), state.EventName.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting event", err.Error())
		return
	}
}

// ImportState accepts "stream_id/event_name" as the composite import ID.
func (r *eventResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: stream_id/event_name, got: %q", req.ID),
		)
		return
	}
	streamID, eventName := parts[0], parts[1]

	event, err := r.client.GetEvent(ctx, streamID, eventName)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Event not found", fmt.Sprintf("No event %q found in stream %q", eventName, streamID))
			return
		}
		resp.Diagnostics.AddError("Error importing event", err.Error())
		return
	}

	var state eventResourceModel
	state.StreamID = types.StringValue(streamID)
	resp.Diagnostics.Append(mapEventToState(ctx, event, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Helpers ---

func mapEventToState(_ context.Context, event *models.Event, state *eventResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(event.EventID)
	state.EventName = types.StringValue(event.EventName)
	state.StreamID = types.StringValue(event.StreamID)
	state.Active = types.BoolValue(event.Active)
	state.Ended = types.BoolValue(event.Ended)

	if event.SubsegmentClipping != nil {
		state.SubsegmentClipping = types.BoolValue(*event.SubsegmentClipping)
	}
	if event.StartTime != nil {
		state.StartTime = types.StringValue(event.StartTime.Format(time.RFC3339))
	} else {
		state.StartTime = types.StringNull()
	}

	if event.EndTime != nil {
		state.EndTime = types.StringValue(event.EndTime.Format(time.RFC3339))
	} else {
		state.EndTime = types.StringNull()
	}

	// egress_filepaths
	fpVals := make([]attr.Value, len(event.EgressFilepaths))
	for i, fp := range event.EgressFilepaths {
		fpVals[i] = types.StringValue(fp)
	}
	state.EgressFilepaths, diags = types.SetValue(types.StringType, fpVals)

	if !event.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(event.CreatedAt.Format(time.RFC3339))
	}
	if !event.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(event.UpdatedAt.Format(time.RFC3339))
	}

	return diags
}
