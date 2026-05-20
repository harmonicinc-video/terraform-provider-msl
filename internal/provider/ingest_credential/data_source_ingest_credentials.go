package ingestcredential

import (
	"context"
	"fmt"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ingestCredentialsDataSource implements datasource.DataSource.
var _ datasource.DataSource = &ingestCredentialsDataSource{}

// NewIngestCredentialsDataSource is the factory function registered with the provider.
func NewIngestCredentialsDataSource() datasource.DataSource {
	return &ingestCredentialsDataSource{}
}

type ingestCredentialsDataSource struct {
	client *client.Client
}

type ingestCredentialsDataSourceModel struct {
	StreamID    types.String `tfsdk:"stream_id"`
	Credentials types.List   `tfsdk:"credentials"`
}

var credentialAttrTypes = map[string]attr.Type{
	"id":          types.StringType,
	"stream_id":   types.StringType,
	"username":    types.StringType,
	"algorithm":   types.StringType,
	"description": types.StringType,
	"expiry_date": types.StringType,
}

func (d *ingestCredentialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingest_credentials"
}

func (d *ingestCredentialsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists MSL5 **Ingest Credentials** (`/api/v1/streams/{stream_id}/ingest_credentials`) for a given stream.\n\n> **Note**: Passwords are never exposed in data source output.",
		Attributes: map[string]schema.Attribute{
			"stream_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the stream to list ingest credentials for.",
			},
			"credentials": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of ingest credentials for the specified stream.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Credential ID (UUID).",
						},
						"stream_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "ID of the parent stream.",
						},
						"username": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Username for ingest authentication.",
						},
						"algorithm": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Hashing algorithm used for the password.",
						},
						"description": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Optional description for the credential.",
						},
						"expiry_date": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "RFC3339 expiry date, or empty string if the credential does not expire.",
						},
					},
				},
			},
		},
	}
}

func (d *ingestCredentialsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ingestCredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ingestCredentialsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds, err := d.client.ListIngestCredentials(ctx, state.StreamID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing ingest credentials", err.Error())
		return
	}

	credObjects := make([]attr.Value, len(creds))
	for i, c := range creds {
		obj, diags := credentialToObject(c, state.StreamID.ValueString())
		resp.Diagnostics.Append(diags...)
		credObjects[i] = obj
	}

	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: credentialAttrTypes}, credObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Credentials = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// credentialToObject converts a models.IngestCredential into the flat types.Object used by the data source list.
func credentialToObject(c models.IngestCredential, streamID string) (types.Object, diag.Diagnostics) {
	description := ""
	if c.Description != nil {
		description = *c.Description
	}
	expiryDate := ""
	if c.ExpiryDate != nil {
		expiryDate = c.ExpiryDate.Format("2006-01-02T15:04:05Z07:00")
	}

	return types.ObjectValue(credentialAttrTypes, map[string]attr.Value{
		"id":          types.StringValue(c.ID),
		"stream_id":   types.StringValue(streamID),
		"username":    types.StringValue(c.Username),
		"algorithm":   types.StringValue(string(c.Algorithm)),
		"description": types.StringValue(description),
		"expiry_date": types.StringValue(expiryDate),
	})
}
