package event

import (
	"context"
	"fmt"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure eventsDataSource implements datasource.DataSource.
var _ datasource.DataSource = &eventsDataSource{}

// NewEventsDataSource is the factory function registered with the provider.
func NewEventsDataSource() datasource.DataSource {
	return &eventsDataSource{}
}

type eventsDataSource struct {
	client *client.Client
}

type eventsDataSourceModel struct {
	StreamID types.String `tfsdk:"stream_id"`
	Events   types.List   `tfsdk:"events"`
}

var eventAttrTypes = map[string]attr.Type{
	"id":                  types.StringType,
	"event_name":          types.StringType,
	"stream_id":           types.StringType,
	"active":              types.BoolType,
	"ended":               types.BoolType,
	"start_time":          types.StringType,
	"end_time":            types.StringType,
	"subsegment_clipping": types.BoolType,
	"created_at":          types.StringType,
	"updated_at":          types.StringType,
}

func (d *eventsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_events"
}

func (d *eventsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists MSL5 **Events** (`/api/v1/streams/{stream_id}/events`) for a given stream.",
		Attributes: map[string]schema.Attribute{
			"stream_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the stream to list events for.",
			},
			"events": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of events for the specified stream.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                  schema.StringAttribute{Computed: true, MarkdownDescription: "Event ID (UUID)."},
						"event_name":          schema.StringAttribute{Computed: true, MarkdownDescription: "Unique event name within the stream."},
						"stream_id":           schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the parent stream."},
						"active":              schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the event is currently active."},
						"ended":               schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the event has ended."},
						"start_time":          schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 start time."},
						"end_time":            schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 end time."},
						"subsegment_clipping": schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether sub-segment clipping is enabled."},
						"created_at":          schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 creation timestamp."},
						"updated_at":          schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 last-updated timestamp."},
					},
				},
			},
		},
	}
}

func (d *eventsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	d.client = c
}

func (d *eventsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state eventsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	events, err := d.client.ListEvents(ctx, state.StreamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing events", err.Error())
		return
	}

	eventObjects := make([]attr.Value, len(events))
	for i, e := range events {
		obj, diags := eventToObject(e)
		resp.Diagnostics.Append(diags...)
		eventObjects[i] = obj
	}

	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: eventAttrTypes}, eventObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Events = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// eventToObject converts a models.Event into the flat types.Object used by the data source list.
func eventToObject(e models.Event) (types.Object, diag.Diagnostics) {
	startTime := ""
	if e.StartTime != nil {
		startTime = e.StartTime.Format(time.RFC3339)
	}
	endTime := ""
	if e.EndTime != nil {
		endTime = e.EndTime.Format(time.RFC3339)
	}
	createdAt := ""
	if !e.CreatedAt.IsZero() {
		createdAt = e.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !e.UpdatedAt.IsZero() {
		updatedAt = e.UpdatedAt.Format(time.RFC3339)
	}
	subsegClipping := false
	if e.SubsegmentClipping != nil {
		subsegClipping = *e.SubsegmentClipping
	}

	return types.ObjectValue(eventAttrTypes, map[string]attr.Value{
		"id":                  types.StringValue(e.EventID),
		"event_name":          types.StringValue(e.EventName),
		"stream_id":           types.StringValue(e.StreamID),
		"active":              types.BoolValue(e.Active),
		"ended":               types.BoolValue(e.Ended),
		"start_time":          types.StringValue(startTime),
		"end_time":            types.StringValue(endTime),
		"subsegment_clipping": types.BoolValue(subsegClipping),
		"created_at":          types.StringValue(createdAt),
		"updated_at":          types.StringValue(updatedAt),
	})
}
