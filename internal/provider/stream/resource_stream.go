// Package stream contains the Terraform resource and data source implementations for MSL streams.
package stream

import (
	"context"
	"fmt"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure streamResource implements required interfaces.
var _ resource.Resource = &streamResource{}
var _ resource.ResourceWithImportState = &streamResource{}

// NewStreamResource is the factory function registered with the provider.
func NewStreamResource() resource.Resource {
	return &streamResource{}
}

type streamResource struct {
	client *client.Client
}

// --- State models ---

type streamResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Description              types.String `tfsdk:"description"`
	Format                   types.String `tfsdk:"format"`
	OriginID                 types.String `tfsdk:"origin_id"`
	IngestLocation           types.String `tfsdk:"ingest_location"`
	BackupIngestLocation     types.String `tfsdk:"backup_ingest_location"`
	ContractID               types.String `tfsdk:"contract_id"`
	CPTag                    types.String `tfsdk:"cptag"`
	GroupID                  types.String `tfsdk:"group_id"`
	PlaylistDurationInMin    types.Int64  `tfsdk:"playlist_duration_in_min"`
	AllowedIPs               types.List   `tfsdk:"allowed_ips"`
	IngestAuthenticationMode types.String `tfsdk:"ingest_authentication_mode"`
	Archiving                types.List   `tfsdk:"archiving"`
	IngestHeader             types.List   `tfsdk:"ingest_header"`
	Playback                 types.List   `tfsdk:"playback"`
	HlsToLlHls               types.List   `tfsdk:"hls_to_llhls"`
	// Computed
	Status               types.String `tfsdk:"status"`
	HostName             types.String `tfsdk:"host_name"`
	BackupHostName       types.String `tfsdk:"backup_host_name"`
	OriginHostName       types.String `tfsdk:"origin_host_name"`
	BackupOriginHostName types.String `tfsdk:"backup_origin_host_name"`
	PrimaryPublishingURL types.String `tfsdk:"primary_publishing_url"`
	BackupPublishingURL  types.String `tfsdk:"backup_publishing_url"`
	CreatedBy            types.String `tfsdk:"created_by"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

type archivingModel struct {
	NoArchive      types.Bool `tfsdk:"no_archive"`
	AutomaticPurge types.List `tfsdk:"automatic_purge"`
}

type automaticPurgeModel struct {
	RetentionDays types.Int64 `tfsdk:"retention_days"`
}

type ingestHeaderModel struct {
	Header types.String `tfsdk:"header"`
	Values types.List   `tfsdk:"values"`
}

type playbackModel struct {
	AkamaiG2oAuth types.List `tfsdk:"akamai_g2o_auth"`
}

type akamaiG2oAuthModel struct {
	Enabled    types.Bool   `tfsdk:"enabled"`
	G2oVersion types.Int64  `tfsdk:"g2o_version"`
	SecretKey  types.String `tfsdk:"secret_key"`
	TimeDelta  types.Int64  `tfsdk:"time_delta"`
}

type hlsToLlHlsModel struct {
	Enabled         types.Bool   `tfsdk:"enabled"`
	SegmentTemplate types.String `tfsdk:"segment_template"`
}

var automaticPurgeAttrTypes = map[string]attr.Type{
	"retention_days": types.Int64Type,
}

var archivingAttrTypes = map[string]attr.Type{
	"no_archive":      types.BoolType,
	"automatic_purge": types.ListType{ElemType: types.ObjectType{AttrTypes: automaticPurgeAttrTypes}},
}

var ingestHeaderAttrTypes = map[string]attr.Type{
	"header": types.StringType,
	"values": types.ListType{ElemType: types.StringType},
}

var akamaiG2oAuthAttrTypes = map[string]attr.Type{
	"enabled":     types.BoolType,
	"g2o_version": types.Int64Type,
	"secret_key":  types.StringType,
	"time_delta":  types.Int64Type,
}

var playbackAttrTypes = map[string]attr.Type{
	"akamai_g2o_auth": types.ListType{ElemType: types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes}},
}

var hlsToLlHlsAttrTypes = map[string]attr.Type{
	"enabled":          types.BoolType,
	"segment_template": types.StringType,
}

// --- Metadata / Schema ---

func (r *streamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stream"
}

func (r *streamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an MSL5 **Stream** resource (`/api/v1/streams`). Streams configure live streaming sessions and depend on an Origin.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique stream ID assigned by the MSL5 API (`stream_id`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Human-readable description of the stream.",
			},
			"format": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Streaming format. One of `HLS`, `CMAF`, `DASH`. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(models.StreamFormatHLS),
						string(models.StreamFormatCMAF),
						string(models.StreamFormatDASH),
					),
				},
			},
			"origin_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the Origin this stream belongs to. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"ingest_location": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Primary ingest zone (e.g. `US_SEA`). Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"backup_ingest_location": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional backup ingest zone. Must differ from `ingest_location`. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"contract_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Akamai contract ID. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"cptag": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "CP code tag. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Group identifier for the stream. Updatable in-place.",
			},
			"playlist_duration_in_min": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(-1),
				MarkdownDescription: "DVR window duration in minutes. Use `-1` to follow the encoder, or `0`–`720` for a fixed window.",
				Validators: []validator.Int64{
					int64validator.Any(
						int64validator.OneOf(-1),
						int64validator.Between(0, 720),
					),
				},
			},
			"allowed_ips": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "List of IP addresses or CIDR blocks allowed to ingest. Empty list means unrestricted. Maximum 270 entries.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(270),
					listvalidator.ValueStringsAre(cidrStringValidator{}),
				},
			},
			"ingest_authentication_mode": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Ingest authentication mode. One of `NONE`, `DIGEST`, `HEADER`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(models.IngestAuthModeNone),
						string(models.IngestAuthModeDigest),
						string(models.IngestAuthModeHeader),
					),
				},
			},
			// Computed fields
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the stream (e.g. `READY`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Primary encoder hostname assigned by MSL5.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"backup_host_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Backup encoder hostname assigned by MSL5.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"origin_host_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The fully-qualified origin hostname derived from the linked origin.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"backup_origin_host_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The fully-qualified backup origin hostname.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"primary_publishing_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Primary publishing URL for this stream.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"backup_publishing_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Backup publishing URL for this stream.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The user who created the stream.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of when the stream was created.",
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
		Blocks: map[string]schema.Block{
			"archiving": schema.ListNestedBlock{
				MarkdownDescription: "Archiving configuration. Required — provide exactly one block.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"no_archive": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "When `true`, archiving is disabled. When `false`, `automatic_purge` is required.",
						},
					},
					Blocks: map[string]schema.Block{
						"automatic_purge": schema.ListNestedBlock{
							MarkdownDescription: "Automatic purge settings. Required when `no_archive` is `false`.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"retention_days": schema.Int64Attribute{
										Required:            true,
										MarkdownDescription: "Number of days to retain archived content (1–62).",
										Validators: []validator.Int64{
											int64validator.Between(1, 62),
										},
									},
								},
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeBetween(1, 1),
				},
			},
			"ingest_header": schema.ListNestedBlock{
				MarkdownDescription: "Ingest header authentication configuration. Used when `ingest_authentication_mode` is `HEADER`.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"header": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "The HTTP header name used for authentication.",
						},
						"values": schema.ListAttribute{
							ElementType:         types.StringType,
							Optional:            true,
							MarkdownDescription: "Accepted header values.",
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"playback": schema.ListNestedBlock{
				MarkdownDescription: "Playback configuration.",
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"akamai_g2o_auth": schema.ListNestedBlock{
							MarkdownDescription: "Akamai G2O playback authentication.",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"enabled": schema.BoolAttribute{
										Optional:            true,
										MarkdownDescription: "Whether G2O authentication is enabled.",
									},
									"g2o_version": schema.Int64Attribute{
										Optional:            true,
										MarkdownDescription: "G2O version number.",
									},
									"secret_key": schema.StringAttribute{
										Optional:            true,
										Sensitive:           true,
										MarkdownDescription: "G2O secret key. This value is sensitive and will not be shown in logs or plan output.",
									},
									"time_delta": schema.Int64Attribute{
										Optional:            true,
										MarkdownDescription: "Allowed time delta in seconds for G2O token validation.",
									},
								},
							},
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"hls_to_llhls": schema.ListNestedBlock{
				MarkdownDescription: "HLS-to-LL-HLS conversion configuration.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"enabled": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Whether HLS-to-LL-HLS conversion is enabled.",
						},
						"segment_template": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Regex pattern to extract the segment number from segment filenames.",
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

// --- Lifecycle ---

func (r *streamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *streamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan streamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, diags := buildCreateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.CreateStream(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating stream", err.Error())
		return
	}

	resp.Diagnostics.Append(mapStreamToState(ctx, stream, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *streamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state streamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.GetStream(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading stream", err.Error())
		return
	}

	resp.Diagnostics.Append(mapStreamToState(ctx, stream, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *streamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan streamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, diags := buildUpdateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stream, err := r.client.UpdateStream(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating stream", err.Error())
		return
	}

	// updated_at is set by the server on every write. UseStateForUnknown() locked the
	// old value in the plan; returning the new API value would trigger Terraform's
	// "inconsistent result after apply" check. Preserve the plan value here — the
	// real new timestamp is refreshed by Read on the next terraform plan.
	savedUpdatedAt := plan.UpdatedAt
	resp.Diagnostics.Append(mapStreamToState(ctx, stream, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.UpdatedAt = savedUpdatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *streamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state streamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteStream(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting stream", err.Error())
		return
	}
}

func (r *streamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var state streamResourceModel
	state.ID = types.StringValue(req.ID)

	stream, err := r.client.GetStream(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Stream not found", fmt.Sprintf("No stream found with ID %q", req.ID))
			return
		}
		resp.Diagnostics.AddError("Error importing stream", err.Error())
		return
	}

	resp.Diagnostics.Append(mapStreamToState(ctx, stream, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Helpers ---

func buildCreateRequest(ctx context.Context, plan streamResourceModel) (models.StreamCreateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	archiving, d := archivingFromState(ctx, plan.Archiving)
	diags.Append(d...)
	if diags.HasError() {
		return models.StreamCreateRequest{}, diags
	}

	req := models.StreamCreateRequest{
		Description:    plan.Description.ValueString(),
		Format:         models.StreamFormat(plan.Format.ValueString()),
		OriginID:       plan.OriginID.ValueString(),
		IngestLocation: plan.IngestLocation.ValueString(),
		ContractID:     plan.ContractID.ValueString(),
		CPTag:          plan.CPTag.ValueString(),
		GroupID:        plan.GroupID.ValueString(),
		Archiving:      archiving,
	}

	if !plan.BackupIngestLocation.IsNull() && !plan.BackupIngestLocation.IsUnknown() {
		v := plan.BackupIngestLocation.ValueString()
		req.BackupIngestLocation = &v
	}
	if !plan.PlaylistDurationInMin.IsNull() && !plan.PlaylistDurationInMin.IsUnknown() {
		v := plan.PlaylistDurationInMin.ValueInt64()
		req.PlaylistDurationInMin = &v
	}
	if !plan.AllowedIPs.IsNull() && !plan.AllowedIPs.IsUnknown() {
		var ips []string
		diags.Append(plan.AllowedIPs.ElementsAs(ctx, &ips, false)...)
		req.AllowedIPs = ips
	}
	if !plan.IngestAuthenticationMode.IsNull() && !plan.IngestAuthenticationMode.IsUnknown() {
		v := models.IngestAuthMode(plan.IngestAuthenticationMode.ValueString())
		req.IngestAuthenticationMode = &v
	}

	req.IngestHeader, d = ingestHeaderFromState(ctx, plan.IngestHeader)
	diags.Append(d...)

	req.Playback, d = playbackFromState(ctx, plan.Playback)
	diags.Append(d...)

	req.HlsToLlHls, d = hlsToLlHlsFromState(ctx, plan.HlsToLlHls)
	diags.Append(d...)

	return req, diags
}

func buildUpdateRequest(ctx context.Context, plan streamResourceModel) (models.StreamUpdateRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	archiving, d := archivingFromState(ctx, plan.Archiving)
	diags.Append(d...)

	req := models.StreamUpdateRequest{
		AllowedIPs: []string{}, // always send (empty = unrestricted); reset semantics
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		req.Description = &v
	}
	if !plan.GroupID.IsNull() && !plan.GroupID.IsUnknown() {
		v := plan.GroupID.ValueString()
		req.GroupID = &v
	}
	req.Archiving = &archiving

	if !plan.PlaylistDurationInMin.IsNull() && !plan.PlaylistDurationInMin.IsUnknown() {
		v := plan.PlaylistDurationInMin.ValueInt64()
		req.PlaylistDurationInMin = &v
	}
	if !plan.AllowedIPs.IsNull() && !plan.AllowedIPs.IsUnknown() {
		var ips []string
		diags.Append(plan.AllowedIPs.ElementsAs(ctx, &ips, false)...)
		req.AllowedIPs = ips
	}
	if !plan.IngestAuthenticationMode.IsNull() && !plan.IngestAuthenticationMode.IsUnknown() {
		v := models.IngestAuthMode(plan.IngestAuthenticationMode.ValueString())
		req.IngestAuthenticationMode = &v
	}

	req.IngestHeader, d = ingestHeaderFromState(ctx, plan.IngestHeader)
	diags.Append(d...)

	req.Playback, d = playbackFromState(ctx, plan.Playback)
	diags.Append(d...)

	req.HlsToLlHls, d = hlsToLlHlsFromState(ctx, plan.HlsToLlHls)
	diags.Append(d...)

	return req, diags
}

func archivingFromState(ctx context.Context, list types.List) (models.Archiving, diag.Diagnostics) {
	var diags diag.Diagnostics
	var archivings []archivingModel
	diags.Append(list.ElementsAs(ctx, &archivings, false)...)
	if diags.HasError() || len(archivings) == 0 {
		return models.Archiving{}, diags
	}

	a := archivings[0]
	result := models.Archiving{NoArchive: a.NoArchive.ValueBool()}

	var purges []automaticPurgeModel
	diags.Append(a.AutomaticPurge.ElementsAs(ctx, &purges, false)...)
	if len(purges) > 0 {
		days := purges[0].RetentionDays.ValueInt64()
		result.AutomaticPurge = &models.AutomaticPurge{RetentionDays: &days}
	}

	return result, diags
}

func ingestHeaderFromState(ctx context.Context, list types.List) (*models.StreamIngestHeader, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var headers []ingestHeaderModel
	diags.Append(list.ElementsAs(ctx, &headers, false)...)
	if diags.HasError() || len(headers) == 0 {
		return nil, diags
	}

	h := headers[0]
	result := &models.StreamIngestHeader{}

	if !h.Header.IsNull() && !h.Header.IsUnknown() {
		v := h.Header.ValueString()
		result.Header = &v
	}
	if !h.Values.IsNull() && !h.Values.IsUnknown() {
		var vals []string
		diags.Append(h.Values.ElementsAs(ctx, &vals, false)...)
		result.Values = vals
	}

	return result, diags
}

func playbackFromState(ctx context.Context, list types.List) (*models.Playback, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var playbacks []playbackModel
	diags.Append(list.ElementsAs(ctx, &playbacks, false)...)
	if diags.HasError() || len(playbacks) == 0 {
		return nil, diags
	}

	pb := playbacks[0]
	result := &models.Playback{}

	var g2oList []akamaiG2oAuthModel
	diags.Append(pb.AkamaiG2oAuth.ElementsAs(ctx, &g2oList, false)...)
	if len(g2oList) > 0 {
		g := g2oList[0]
		g2o := &models.AkamaiG2oAuth{}
		if !g.Enabled.IsNull() && !g.Enabled.IsUnknown() {
			v := g.Enabled.ValueBool()
			g2o.Enabled = &v
		}
		if !g.G2oVersion.IsNull() && !g.G2oVersion.IsUnknown() {
			v := g.G2oVersion.ValueInt64()
			g2o.G2oVersion = &v
		}
		if !g.SecretKey.IsNull() && !g.SecretKey.IsUnknown() {
			v := g.SecretKey.ValueString()
			g2o.SecretKey = &v
		}
		if !g.TimeDelta.IsNull() && !g.TimeDelta.IsUnknown() {
			v := g.TimeDelta.ValueInt64()
			g2o.TimeDelta = &v
		}
		result.AkamaiG2oAuth = g2o
	}

	return result, diags
}

func hlsToLlHlsFromState(ctx context.Context, list types.List) (*models.HlsToLlHlsConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var cfgs []hlsToLlHlsModel
	diags.Append(list.ElementsAs(ctx, &cfgs, false)...)
	if diags.HasError() || len(cfgs) == 0 {
		return nil, diags
	}

	cfg := cfgs[0]
	return &models.HlsToLlHlsConfig{
		Enabled:         cfg.Enabled.ValueBool(),
		SegmentTemplate: cfg.SegmentTemplate.ValueString(),
	}, diags
}

// mapStreamToState populates the Terraform state from an API Stream struct.
func mapStreamToState(ctx context.Context, stream *models.Stream, state *streamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(stream.ID)
	state.Description = types.StringValue(stream.Description)
	state.Format = types.StringValue(string(stream.Format))
	state.OriginID = types.StringValue(stream.OriginID)
	state.IngestLocation = types.StringValue(stream.IngestLocation)
	state.ContractID = types.StringValue(stream.ContractID)
	state.CPTag = types.StringValue(stream.CPTag)
	state.GroupID = types.StringValue(stream.GroupID)
	state.Status = types.StringValue(string(stream.Status))
	state.HostName = types.StringValue(stream.HostName)
	state.OriginHostName = types.StringValue(stream.OriginHostName)
	state.PrimaryPublishingURL = types.StringValue(stream.PrimaryPublishingURL)
	state.BackupPublishingURL = types.StringValue(stream.BackupPublishingURL)
	state.CreatedBy = types.StringValue(stream.CreatedBy)

	if stream.BackupIngestLocation != nil {
		state.BackupIngestLocation = types.StringValue(*stream.BackupIngestLocation)
	} else if state.BackupIngestLocation.IsUnknown() {
		state.BackupIngestLocation = types.StringNull()
	}

	if stream.BackupHostName != nil {
		state.BackupHostName = types.StringValue(*stream.BackupHostName)
	} else {
		state.BackupHostName = types.StringNull()
	}

	if stream.BackupOriginHostName != nil {
		state.BackupOriginHostName = types.StringValue(*stream.BackupOriginHostName)
	} else {
		state.BackupOriginHostName = types.StringNull()
	}

	if stream.PlaylistDurationInMin != nil {
		state.PlaylistDurationInMin = types.Int64Value(*stream.PlaylistDurationInMin)
	} else {
		state.PlaylistDurationInMin = types.Int64Value(-1)
	}

	// allowed_ips
	if len(stream.AllowedIPs) > 0 {
		ipVals := make([]attr.Value, len(stream.AllowedIPs))
		for i, ip := range stream.AllowedIPs {
			ipVals[i] = types.StringValue(ip)
		}
		state.AllowedIPs, diags = types.ListValue(types.StringType, ipVals)
	} else {
		state.AllowedIPs = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// ingest_authentication_mode: map the API response back to state.
	modeStr := ""
	if stream.IngestAuthenticationMode != nil {
		modeStr = string(*stream.IngestAuthenticationMode)
	}
	switch {
	case modeStr != "" && modeStr != string(models.IngestAuthModeNone):
		state.IngestAuthenticationMode = types.StringValue(modeStr)
	case modeStr == string(models.IngestAuthModeNone):
		// API explicitly returned NONE — write it to state so any previous non-NONE
		// value is overwritten rather than left stale.
		state.IngestAuthenticationMode = types.StringValue(string(models.IngestAuthModeNone))
	case stream.IngestAuthentication != nil && *stream.IngestAuthentication:
		// Legacy API field returned — translate to the equivalent mode.
		state.IngestAuthenticationMode = types.StringValue(string(models.IngestAuthModeDigest))
	default:
		// Both fields absent/nil — preserve existing state value (null if never set).
	}

	// ingest_header
	diags.Append(mapIngestHeaderToState(stream, state)...)

	// playback
	diags.Append(mapPlaybackToState(ctx, stream, state)...)

	// archiving
	diags.Append(mapArchivingToState(stream, state)...)

	// hls_to_llhls
	if stream.HlsToLlHls != nil {
		hlsObj, d := types.ObjectValue(hlsToLlHlsAttrTypes, map[string]attr.Value{
			"enabled":          types.BoolValue(stream.HlsToLlHls.Enabled),
			"segment_template": types.StringValue(stream.HlsToLlHls.SegmentTemplate),
		})
		diags.Append(d...)
		state.HlsToLlHls, d = types.ListValue(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{hlsObj})
		diags.Append(d...)
	} else {
		state.HlsToLlHls = types.ListValueMust(types.ObjectType{AttrTypes: hlsToLlHlsAttrTypes}, []attr.Value{})
	}

	if !stream.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(stream.CreatedAt.Format(time.RFC3339))
	}
	if !stream.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(stream.UpdatedAt.Format(time.RFC3339))
	}

	return diags
}

func mapIngestHeaderToState(stream *models.Stream, state *streamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if stream.IngestHeader != nil {
		headerVal := types.StringNull()
		if stream.IngestHeader.Header != nil {
			headerVal = types.StringValue(*stream.IngestHeader.Header)
		}
		valVals := make([]attr.Value, len(stream.IngestHeader.Values))
		for i, v := range stream.IngestHeader.Values {
			valVals[i] = types.StringValue(v)
		}
		valsList, d := types.ListValue(types.StringType, valVals)
		diags.Append(d...)
		ihObj, d := types.ObjectValue(ingestHeaderAttrTypes, map[string]attr.Value{
			"header": headerVal,
			"values": valsList,
		})
		diags.Append(d...)
		state.IngestHeader, d = types.ListValue(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{ihObj})
		diags.Append(d...)
	} else {
		state.IngestHeader = types.ListValueMust(types.ObjectType{AttrTypes: ingestHeaderAttrTypes}, []attr.Value{})
	}
	return diags
}

func mapPlaybackToState(ctx context.Context, stream *models.Stream, state *streamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if stream.Playback == nil {
		state.Playback = types.ListValueMust(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{})
		return diags
	}

	g2oList := types.ListValueMust(types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes}, []attr.Value{})
	if stream.Playback.AkamaiG2oAuth != nil {
		g2oList, diags = mapG2oAuthToState(ctx, stream.Playback.AkamaiG2oAuth, state)
	}
	pbObj, d := types.ObjectValue(playbackAttrTypes, map[string]attr.Value{
		"akamai_g2o_auth": g2oList,
	})
	diags.Append(d...)
	state.Playback, d = types.ListValue(types.ObjectType{AttrTypes: playbackAttrTypes}, []attr.Value{pbObj})
	diags.Append(d...)
	return diags
}

func mapG2oAuthToState(ctx context.Context, g *models.AkamaiG2oAuth, state *streamResourceModel) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	enabledVal := types.BoolNull()
	if g.Enabled != nil {
		enabledVal = types.BoolValue(*g.Enabled)
	}
	g2oVersionVal := types.Int64Null()
	if g.G2oVersion != nil {
		g2oVersionVal = types.Int64Value(*g.G2oVersion)
	}
	// secret_key is never returned by the API — preserve the state value
	secretKeyVal := types.StringNull()
	if !state.Playback.IsNull() && !state.Playback.IsUnknown() {
		var pbs []playbackModel
		if d := state.Playback.ElementsAs(ctx, &pbs, false); !d.HasError() && len(pbs) > 0 {
			var g2os []akamaiG2oAuthModel
			if d := pbs[0].AkamaiG2oAuth.ElementsAs(ctx, &g2os, false); !d.HasError() && len(g2os) > 0 {
				secretKeyVal = g2os[0].SecretKey
			}
		}
	}
	timeDeltaVal := types.Int64Null()
	if g.TimeDelta != nil {
		timeDeltaVal = types.Int64Value(*g.TimeDelta)
	}
	g2oObj, d := types.ObjectValue(akamaiG2oAuthAttrTypes, map[string]attr.Value{
		"enabled":     enabledVal,
		"g2o_version": g2oVersionVal,
		"secret_key":  secretKeyVal,
		"time_delta":  timeDeltaVal,
	})
	diags.Append(d...)
	g2oList, d := types.ListValue(types.ObjectType{AttrTypes: akamaiG2oAuthAttrTypes}, []attr.Value{g2oObj})
	diags.Append(d...)
	return g2oList, diags
}

func mapArchivingToState(stream *models.Stream, state *streamResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if stream.Archiving == nil {
		return diags
	}
	purgeList := types.ListValueMust(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{})
	if stream.Archiving.AutomaticPurge != nil && stream.Archiving.AutomaticPurge.RetentionDays != nil {
		purgeObj, d := types.ObjectValue(automaticPurgeAttrTypes, map[string]attr.Value{
			"retention_days": types.Int64Value(*stream.Archiving.AutomaticPurge.RetentionDays),
		})
		diags.Append(d...)
		purgeList, d = types.ListValue(types.ObjectType{AttrTypes: automaticPurgeAttrTypes}, []attr.Value{purgeObj})
		diags.Append(d...)
	}
	archObj, d := types.ObjectValue(archivingAttrTypes, map[string]attr.Value{
		"no_archive":      types.BoolValue(stream.Archiving.NoArchive),
		"automatic_purge": purgeList,
	})
	diags.Append(d...)
	state.Archiving, d = types.ListValue(types.ObjectType{AttrTypes: archivingAttrTypes}, []attr.Value{archObj})
	diags.Append(d...)
	return diags
}
