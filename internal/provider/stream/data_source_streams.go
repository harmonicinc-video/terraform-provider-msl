package stream

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

// Ensure streamsDataSource implements datasource.DataSource.
var _ datasource.DataSource = &streamsDataSource{}

// NewStreamsDataSource is the factory function registered with the provider.
func NewStreamsDataSource() datasource.DataSource {
	return &streamsDataSource{}
}

type streamsDataSource struct {
	client *client.Client
}

type streamsDataSourceModel struct {
	ContractID types.String `tfsdk:"contract_id"`
	GroupID    types.String `tfsdk:"group_id"`
	Streams    types.List   `tfsdk:"streams"`
}

var streamAttrTypes = map[string]attr.Type{
	"id":                       types.StringType,
	"description":              types.StringType,
	"format":                   types.StringType,
	"status":                   types.StringType,
	"origin_id":                types.StringType,
	"ingest_location":          types.StringType,
	"backup_ingest_location":   types.StringType,
	"contract_id":              types.StringType,
	"cptag":                    types.StringType,
	"group_id":                 types.StringType,
	"host_name":                types.StringType,
	"primary_publishing_url":   types.StringType,
	"backup_publishing_url":    types.StringType,
	"playlist_duration_in_min": types.Int64Type,
	"created_by":               types.StringType,
	"created_at":               types.StringType,
	"updated_at":               types.StringType,
}

func (d *streamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_streams"
}

func (d *streamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists MSL5 **Streams** (`/api/v1/streams`), optionally filtered by contract and group.",
		Attributes: map[string]schema.Attribute{
			"contract_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter streams by Akamai contract ID.",
			},
			"group_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter streams by group ID.",
			},
			"streams": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of streams matching the specified filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                       schema.StringAttribute{Computed: true, MarkdownDescription: "Stream ID."},
						"description":              schema.StringAttribute{Computed: true, MarkdownDescription: "Stream description."},
						"format":                   schema.StringAttribute{Computed: true, MarkdownDescription: "Streaming format (HLS, CMAF, DASH)."},
						"status":                   schema.StringAttribute{Computed: true, MarkdownDescription: "Current stream status."},
						"origin_id":                schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the linked origin."},
						"ingest_location":          schema.StringAttribute{Computed: true, MarkdownDescription: "Primary ingest zone."},
						"backup_ingest_location":   schema.StringAttribute{Computed: true, MarkdownDescription: "Backup ingest zone."},
						"contract_id":              schema.StringAttribute{Computed: true, MarkdownDescription: "Akamai contract ID."},
						"cptag":                    schema.StringAttribute{Computed: true, MarkdownDescription: "CP code tag."},
						"group_id":                 schema.StringAttribute{Computed: true, MarkdownDescription: "Group identifier."},
						"host_name":                schema.StringAttribute{Computed: true, MarkdownDescription: "Primary encoder hostname."},
						"primary_publishing_url":   schema.StringAttribute{Computed: true, MarkdownDescription: "Primary publishing URL."},
						"backup_publishing_url":    schema.StringAttribute{Computed: true, MarkdownDescription: "Backup publishing URL."},
						"playlist_duration_in_min": schema.Int64Attribute{Computed: true, MarkdownDescription: "DVR window in minutes."},
						"created_by":               schema.StringAttribute{Computed: true, MarkdownDescription: "User who created the stream."},
						"created_at":               schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 creation timestamp."},
						"updated_at":               schema.StringAttribute{Computed: true, MarkdownDescription: "RFC3339 last-updated timestamp."},
					},
				},
			},
		},
	}
}

func (d *streamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *streamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state streamsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contractID := ""
	if !state.ContractID.IsNull() && !state.ContractID.IsUnknown() {
		contractID = state.ContractID.ValueString()
	}
	groupID := ""
	if !state.GroupID.IsNull() && !state.GroupID.IsUnknown() {
		groupID = state.GroupID.ValueString()
	}

	streams, err := d.client.ListStreams(ctx, contractID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing streams", err.Error())
		return
	}

	streamObjects := make([]attr.Value, len(streams))
	for i, s := range streams {
		obj, diags := streamSummaryToObject(s)
		resp.Diagnostics.Append(diags...)
		streamObjects[i] = obj
	}

	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: streamAttrTypes}, streamObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Streams = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// streamSummaryToObject converts a models.Stream into the flat types.Object used by the data source list.
func streamSummaryToObject(s models.Stream) (types.Object, diag.Diagnostics) {
	createdAt := ""
	if !s.CreatedAt.IsZero() {
		createdAt = s.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !s.UpdatedAt.IsZero() {
		updatedAt = s.UpdatedAt.Format(time.RFC3339)
	}
	backupIngestLocation := ""
	if s.BackupIngestLocation != nil {
		backupIngestLocation = *s.BackupIngestLocation
	}
	playlistDur := int64(-1)
	if s.PlaylistDurationInMin != nil {
		playlistDur = *s.PlaylistDurationInMin
	}

	return types.ObjectValue(streamAttrTypes, map[string]attr.Value{
		"id":                       types.StringValue(s.ID),
		"description":              types.StringValue(s.Description),
		"format":                   types.StringValue(string(s.Format)),
		"status":                   types.StringValue(string(s.Status)),
		"origin_id":                types.StringValue(s.OriginID),
		"ingest_location":          types.StringValue(s.IngestLocation),
		"backup_ingest_location":   types.StringValue(backupIngestLocation),
		"contract_id":              types.StringValue(s.ContractID),
		"cptag":                    types.StringValue(s.CPTag),
		"group_id":                 types.StringValue(s.GroupID),
		"host_name":                types.StringValue(s.HostName),
		"primary_publishing_url":   types.StringValue(s.PrimaryPublishingURL),
		"backup_publishing_url":    types.StringValue(s.BackupPublishingURL),
		"playlist_duration_in_min": types.Int64Value(playlistDur),
		"created_by":               types.StringValue(s.CreatedBy),
		"created_at":               types.StringValue(createdAt),
		"updated_at":               types.StringValue(updatedAt),
	})
}
