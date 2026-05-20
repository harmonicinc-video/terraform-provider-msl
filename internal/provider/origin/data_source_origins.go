// Package origin contains the Terraform resource and data source implementations for MSL origins.
package origin

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

// Ensure originsDataSource implements datasource.DataSource.
var _ datasource.DataSource = &originsDataSource{}

// NewOriginsDataSource is the factory function registered with the provider.
func NewOriginsDataSource() datasource.DataSource {
	return &originsDataSource{}
}

type originsDataSource struct {
	client *client.Client
}

// originsDataSourceModel is the top-level data source model.
type originsDataSourceModel struct {
	ContractID types.String `tfsdk:"contract_id"`
	GroupID    types.String `tfsdk:"group_id"`
	Origins    types.List   `tfsdk:"origins"`
}

var originAttrTypes = map[string]attr.Type{
	"id":                     types.StringType,
	"account_id":             types.StringType,
	"host_name":              types.StringType,
	"ingest_location":        types.StringType,
	"backup_ingest_location": types.StringType,
	"contract_id":            types.StringType,
	"cptag":                  types.StringType,
	"group_id":               types.StringType,
	"status":                 types.StringType,
	"created_at":             types.StringType,
	"updated_at":             types.StringType,
	"created_by":             types.StringType,
}

func (d *originsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_origins"
}

func (d *originsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists MSL5 **Origins** (`/api/v1/origins`), optionally filtered by contract and group.",
		Attributes: map[string]schema.Attribute{
			"contract_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter origins by Akamai contract ID.",
			},
			"group_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter origins by group ID.",
			},
			"origins": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of origins matching the specified filters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Origin ID.",
						}, "account_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Account ID associated with the origin.",
						}, "host_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Unique hostname identifier.",
						},
						"ingest_location": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Primary ingest zone.",
						},
						"backup_ingest_location": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Backup ingest zone (empty string if not set).",
						},
						"contract_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Akamai contract ID.",
						},
						"cptag": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "CP code tag.",
						},
						"group_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Group identifier.",
						},
						"status": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Current status of the origin.",
						},
						"created_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "RFC3339 creation timestamp.",
						},
						"updated_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "RFC3339 last-updated timestamp.",
						},
						"created_by": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User who created the origin.",
						},
					},
				},
			},
		},
	}
}

func (d *originsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *originsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state originsDataSourceModel
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

	origins, err := d.client.ListOrigins(ctx, contractID, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing origins", err.Error())
		return
	}

	originObjects := make([]attr.Value, len(origins))
	for i, o := range origins {
		obj, diags := originSummaryToObject(o)
		resp.Diagnostics.Append(diags...)
		originObjects[i] = obj
	}

	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: originAttrTypes}, originObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Origins = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// originSummaryToObject converts a models.Origin into the flat types.Object used by the data source list.
func originSummaryToObject(o models.Origin) (types.Object, diag.Diagnostics) {
	createdAt := ""
	if !o.CreatedAt.IsZero() {
		createdAt = o.CreatedAt.Format(time.RFC3339)
	}
	updatedAt := ""
	if !o.UpdatedAt.IsZero() {
		updatedAt = o.UpdatedAt.Format(time.RFC3339)
	}

	return types.ObjectValue(originAttrTypes, map[string]attr.Value{
		"id":                     types.StringValue(o.ID),
		"account_id":             types.StringValue(o.AccountID),
		"host_name":              types.StringValue(o.HostName),
		"ingest_location":        types.StringValue(o.IngestLocation),
		"backup_ingest_location": types.StringValue(o.BackupIngestLocation),
		"contract_id":            types.StringValue(o.ContractID),
		"cptag":                  types.StringValue(o.CPTag),
		"group_id":               types.StringValue(o.GroupID),
		"status":                 types.StringValue(string(o.Status)),
		"created_at":             types.StringValue(createdAt),
		"updated_at":             types.StringValue(updatedAt),
		"created_by":             types.StringValue(o.CreatedBy),
	})
}
