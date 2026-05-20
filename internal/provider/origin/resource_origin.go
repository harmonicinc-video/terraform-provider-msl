package origin

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// hostNameMatchesFQDN reports whether fqdn is the API-assigned FQDN for the
// given short hostName and optional ingestLocation.
//
// The MSL5 API prepends the ingest_location as a lower-kebab-case prefix:
//
//	ingest_location="US_SEA", host_name="myhost"
//	→ "us-sea-myhost.dev.mslorigin.nebula.video"
//
// Two patterns are checked:
//  1. fqdn starts with "{hostName}."              (location-less form)
//  2. fqdn starts with "{locationPrefix}-{hostName}."
func hostNameMatchesFQDN(fqdn, hostName, ingestLocation string) bool {
	if strings.HasPrefix(fqdn, hostName+".") {
		return true
	}
	if ingestLocation != "" {
		locationPrefix := strings.ToLower(strings.ReplaceAll(ingestLocation, "_", "-"))
		if strings.HasPrefix(fqdn, locationPrefix+"-"+hostName+".") {
			return true
		}
	}
	return false
}

// hostNameFQDNModifier suppresses diff that occurs after
// "terraform import" when the API returns a fully-qualified domain name but
// the configuration contains only the short name.
//
// The MSL5 API prepends the ingest_location as a lower-kebab-case prefix:
//
//	ingest_location="US_SEA", host_name="myhost"
//	→ API returns "us-sea-myhost.dev.mslorigin.nebula.video"
//
// The modifier reads the sibling location attribute from the plan (controlled by
// locationAttr; defaults to "ingest_location") and checks two patterns:
//  1. stateVal starts with "{planHostName}."
//  2. stateVal starts with "{locationPrefix}-{planHostName}."
//
// When either matches, the plan value is normalised to the stored FQDN so
// that the subsequent modifier sees no change.
type hostNameFQDNModifier struct {
	// locationAttr is the plan attribute name that holds the ingest location
	// (e.g. "ingest_location" or "backup_ingest_location").
	// Defaults to "ingest_location" when empty.
	locationAttr string
}

func (m hostNameFQDNModifier) Description(_ context.Context) string {
	return "Suppresses replacement when the config short name matches the API-assigned FQDN stored after import."
}

func (m hostNameFQDNModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m hostNameFQDNModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	planVal := req.PlanValue.ValueString()
	stateVal := req.StateValue.ValueString()

	// Pattern 1: FQDN is just "{hostname}.{domain}" — no location needed.
	if strings.HasPrefix(stateVal, planVal+".") {
		resp.PlanValue = req.StateValue
		return
	}

	// Pattern 2: FQDN is "{location_prefix}-{hostname}.{domain}"
	// Requires reading the location attribute from the plan.
	locAttr := m.locationAttr
	if locAttr == "" {
		locAttr = "ingest_location"
	}
	var ingestLoc types.String
	if diags := req.Plan.GetAttribute(ctx, path.Root(locAttr), &ingestLoc); diags.HasError() {
		return
	}
	if !ingestLoc.IsNull() && !ingestLoc.IsUnknown() {
		locationPrefix := strings.ToLower(strings.ReplaceAll(ingestLoc.ValueString(), "_", "-"))
		if strings.HasPrefix(stateVal, locationPrefix+"-"+planVal+".") {
			resp.PlanValue = req.StateValue
		}
	}
}

// Ensure originResource implements required interfaces.
var _ resource.Resource = &originResource{}
var _ resource.ResourceWithImportState = &originResource{}
var _ resource.ResourceWithValidateConfig = &originResource{}

// NewOriginResource is the factory function registered with the provider.
func NewOriginResource() resource.Resource {
	return &originResource{}
}

type originResource struct {
	client *client.Client
}

// --- Model ---

// originResourceModel maps to the Terraform state for an msl_origin resource.
type originResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	AccountID            types.String `tfsdk:"account_id"`
	HostName             types.String `tfsdk:"host_name"`
	IngestLocation       types.String `tfsdk:"ingest_location"`
	BackupIngestLocation types.String `tfsdk:"backup_ingest_location"`
	ContractID           types.String `tfsdk:"contract_id"`
	CPTag                types.String `tfsdk:"cptag"`
	GroupID              types.String `tfsdk:"group_id"`
	SharedKeys           types.List   `tfsdk:"shared_keys"`
	Status               types.String `tfsdk:"status"`
	BackupHostName       types.String `tfsdk:"backup_host_name"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
	CreatedBy            types.String `tfsdk:"created_by"`
}

// sharedKeyModel maps a single shared_keys block entry.
type sharedKeyModel struct {
	Name     types.String `tfsdk:"name"`
	Key      types.String `tfsdk:"key"`
	HostName types.String `tfsdk:"host_name"`
	Enabled  types.Bool   `tfsdk:"enabled"`
}

var sharedKeyAttrTypes = map[string]attr.Type{
	"name":      types.StringType,
	"key":       types.StringType,
	"host_name": types.StringType,
	"enabled":   types.BoolType,
}

// --- Metadata / Schema ---

func (r *originResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_origin"
}

func (r *originResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an MSL5 **Origin** resource (`/api/v1/origins`). Origins define the ingest server location and configuration for live streaming.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique origin ID assigned by the MSL5 API (`origin_id`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The account ID associated with the origin, as returned by the MSL5 API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique hostname identifier for this origin. Must be non-empty and unique across the system. Immutable: can only be set during resource creation.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					hostNameFQDNModifier{},
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"ingest_location": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Primary ingest zone in `REGION_ZONE` format (e.g. `US_ORD`, `US_SEA`, `GB_LON`). Must be non-empty. Immutable: can only be set during resource creation.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Z]{2,3}_[A-Z]{2,5}$`),
						"must be a valid ingest zone (e.g. US_ORD, US_SEA, GB_LON)",
					),
				},
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"backup_ingest_location": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Optional backup ingest zone in `REGION_ZONE` format (e.g. `US_SEA`, `GB_LON`). Must differ from `ingest_location`. Immutable: can only be set during resource creation.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Z]{2,3}_[A-Z]{2,5}$`),
						"must be a valid ingest zone (e.g. US_ORD, US_SEA, GB_LON)",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"contract_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Akamai contract ID (e.g. `A-123456`). Immutable: can only be set during resource creation.",
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
				MarkdownDescription: "Group identifier for the origin. Updatable in-place.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the origin (e.g. `READY`).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"backup_host_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Backup encoder hostname assigned by MSL5.",
				PlanModifiers: []planmodifier.String{
					hostNameFQDNModifier{locationAttr: "backup_ingest_location"},
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC3339 timestamp of when the origin was created.",
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
			"created_by": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The user who created the origin.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"shared_keys": schema.ListNestedBlock{
				MarkdownDescription: "Up to 10 shared keys for token-authenticated ingest.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Descriptive name of this shared key entry. Must be between 1 and 8 characters.",
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 8),
							},
						},
						"key": schema.StringAttribute{
							Required:            true,
							Sensitive:           true,
							MarkdownDescription: "The shared key secret value.",
						},
						"host_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Hostname that this shared key applies to.",
						},
						"enabled": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(true),
							MarkdownDescription: "Whether this shared key is active. Defaults to `true`.",
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(10),
				},
			},
		},
	}
}

// --- Lifecycle ---

func (r *originResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config originResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateBackupLocation(config)...)
}

func (r *originResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *originResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan originResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := validateBackupLocation(plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	apiReq := models.OriginCreateRequest{
		HostName:       plan.HostName.ValueString(),
		IngestLocation: plan.IngestLocation.ValueString(),
		ContractID:     plan.ContractID.ValueString(),
		CPTag:          plan.CPTag.ValueString(),
		GroupID:        plan.GroupID.ValueString(),
	}
	if !plan.BackupIngestLocation.IsNull() && !plan.BackupIngestLocation.IsUnknown() {
		apiReq.BackupIngestLocation = plan.BackupIngestLocation.ValueString()
	}

	apiReq.SharedKeys, resp.Diagnostics = sharedKeysFromState(ctx, plan.SharedKeys)
	if resp.Diagnostics.HasError() {
		return
	}

	origin, err := r.client.CreateOrigin(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating origin", err.Error())
		return
	}

	resp.Diagnostics.Append(mapOriginToState(ctx, origin, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *originResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state originResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	origin, err := r.client.GetOrigin(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading origin", err.Error())
		return
	}

	resp.Diagnostics.Append(mapOriginToState(ctx, origin, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *originResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan originResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := models.OriginUpdateRequest{
		GroupID: plan.GroupID.ValueString(),
	}
	var diags diag.Diagnostics
	apiReq.SharedKeys, diags = sharedKeysFromState(ctx, plan.SharedKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	origin, err := r.client.UpdateOrigin(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating origin", err.Error())
		return
	}

	// updated_at is set by the server on every write. UseStateForUnknown() locked the
	// old value in the plan; returning the new API value would trigger Terraform's
	// "inconsistent result after apply" check. Preserve the plan value here — the
	// real new timestamp is refreshed by Read on the next terraform plan.
	savedUpdatedAt := plan.UpdatedAt
	resp.Diagnostics.Append(mapOriginToState(ctx, origin, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.UpdatedAt = savedUpdatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *originResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state originResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if diags := validateDeleteStatus(state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if err := r.client.DeleteOrigin(ctx, state.ID.ValueString()); err != nil {
		apiErr, ok := err.(*client.APIError)
		if ok && apiErr.StatusCode == 409 {
			resp.Diagnostics.AddError(
				"Cannot delete origin with active streams",
				fmt.Sprintf("The origin %q has active streams. Delete all streams associated with this origin before removing it.\n\nAPI response: %s", state.ID.ValueString(), apiErr.Body),
			)
			return
		}
		resp.Diagnostics.AddError("Error deleting origin", err.Error())
		return
	}
}

func (r *originResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by origin_id
	var state originResourceModel
	state.ID = types.StringValue(req.ID)

	origin, err := r.client.GetOrigin(ctx, req.ID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Origin not found", fmt.Sprintf("No origin found with ID %q", req.ID))
			return
		}
		resp.Diagnostics.AddError("Error importing origin", err.Error())
		return
	}

	resp.Diagnostics.Append(mapOriginToState(ctx, origin, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Helpers ---

// validateDeleteStatus returns an error diagnostic if the origin is not in a
// state that permits deletion. Only READY origins can be deleted.
func validateDeleteStatus(state originResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if state.Status.ValueString() != string(models.OriginStatusReady) {
		diags.AddError(
			"Cannot delete origin",
			fmt.Sprintf("Origin %q has status %q. Only origins with status READY can be deleted.", state.ID.ValueString(), state.Status.ValueString()),
		)
	}
	return diags
}

func validateBackupLocation(plan originResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if !plan.BackupIngestLocation.IsNull() && !plan.BackupIngestLocation.IsUnknown() &&
		plan.BackupIngestLocation.ValueString() == plan.IngestLocation.ValueString() {
		diags.AddAttributeError(
			path.Root("backup_ingest_location"),
			"Invalid backup ingest location",
			"backup_ingest_location must differ from ingest_location.",
		)
	}
	return diags
}

// sharedKeysFromState converts a types.List of shared key objects into the client model.
func sharedKeysFromState(ctx context.Context, list types.List) ([]models.SharedKey, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var keyModels []sharedKeyModel
	diags.Append(list.ElementsAs(ctx, &keyModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	keys := make([]models.SharedKey, len(keyModels))
	for i, m := range keyModels {
		keys[i] = models.SharedKey{
			Name:     m.Name.ValueString(),
			Key:      m.Key.ValueString(),
			HostName: m.HostName.ValueString(),
			Enabled:  m.Enabled.ValueBool(),
		}
	}
	return keys, diags
}

// mapOriginToState populates the Terraform state model from an API Origin struct.
// host_name is intentionally not overwritten when it already has a value: the API
// server transforms the user-supplied short name into a FQDN, so we preserve the
// original config value to avoid a "provider produced inconsistent result" error.
func mapOriginToState(_ context.Context, origin *models.Origin, state *originResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(origin.ID)
	state.AccountID = types.StringValue(origin.AccountID)
	// Preserve user-provided host_name unless we are populating from scratch (e.g. import).
	if state.HostName.IsNull() || state.HostName.IsUnknown() {
		state.HostName = types.StringValue(origin.HostName)
	}
	state.IngestLocation = types.StringValue(origin.IngestLocation)
	state.ContractID = types.StringValue(origin.ContractID)
	state.CPTag = types.StringValue(origin.CPTag)
	state.GroupID = types.StringValue(origin.GroupID)
	state.Status = types.StringValue(string(origin.Status))
	// Preserve backup_host_name unless populating from scratch (e.g. import).
	// The API assigns an FQDN; keeping the initially stored value avoids a
	// spurious diff on the next plan.
	if state.BackupHostName.IsNull() || state.BackupHostName.IsUnknown() {
		if origin.BackupHostName != "" {
			state.BackupHostName = types.StringValue(origin.BackupHostName)
		} else {
			state.BackupHostName = types.StringNull()
		}
	}
	state.CreatedBy = types.StringValue(origin.CreatedBy)

	// Preserve user-provided backup_ingest_location unless populating from
	// scratch (e.g. import), for the same reason as host_name.
	if state.BackupIngestLocation.IsNull() || state.BackupIngestLocation.IsUnknown() {
		if origin.BackupIngestLocation != "" {
			state.BackupIngestLocation = types.StringValue(origin.BackupIngestLocation)
		} else {
			state.BackupIngestLocation = types.StringNull()
		}
	}

	if !origin.CreatedAt.IsZero() {
		state.CreatedAt = types.StringValue(origin.CreatedAt.Format(time.RFC3339))
	}
	if !origin.UpdatedAt.IsZero() {
		state.UpdatedAt = types.StringValue(origin.UpdatedAt.Format(time.RFC3339))
	}

	// Map shared keys
	if len(origin.SharedKeys) == 0 {
		state.SharedKeys = types.ListValueMust(types.ObjectType{AttrTypes: sharedKeyAttrTypes}, []attr.Value{})
	} else {
		keyObjects := make([]attr.Value, len(origin.SharedKeys))
		for i, sk := range origin.SharedKeys {
			obj, d := types.ObjectValue(sharedKeyAttrTypes, map[string]attr.Value{
				"name":      types.StringValue(sk.Name),
				"key":       types.StringValue(sk.Key),
				"host_name": types.StringValue(sk.HostName),
				"enabled":   types.BoolValue(sk.Enabled),
			})
			diags.Append(d...)
			keyObjects[i] = obj
		}
		state.SharedKeys, diags = types.ListValue(types.ObjectType{AttrTypes: sharedKeyAttrTypes}, keyObjects)
	}

	return diags
}
