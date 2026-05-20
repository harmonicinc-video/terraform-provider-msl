// Package ingestcredential contains the Terraform resource and data source for MSL ingest credentials.
package ingestcredential

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/planmodifiers"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure ingestCredentialResource implements required interfaces.
var _ resource.Resource = &ingestCredentialResource{}
var _ resource.ResourceWithImportState = &ingestCredentialResource{}

// NewIngestCredentialResource is the factory function registered with the provider.
func NewIngestCredentialResource() resource.Resource {
	return &ingestCredentialResource{}
}

type ingestCredentialResource struct {
	client *client.Client
}

// --- State model ---

type ingestCredentialResourceModel struct {
	ID          types.String `tfsdk:"id"`
	StreamID    types.String `tfsdk:"stream_id"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
	Algorithm   types.String `tfsdk:"algorithm"`
	Description types.String `tfsdk:"description"`
	ExpiryDate  types.String `tfsdk:"expiry_date"`
}

// --- Metadata / Schema ---

func (r *ingestCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ingest_credential"
}

func (r *ingestCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an MSL5 **Ingest Credential** (`/api/v1/streams/{stream_id}/ingest_credentials`). Passwords are write-only and are never returned by the API. A maximum of 4 credentials per stream is supported.\n\n> **Note**: When importing a credential, the `password` attribute will be unknown. You must add it manually in your configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The unique credential ID assigned by the MSL5 API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"stream_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the stream this credential belongs to. Immutable: can only be set during resource creation.",
				PlanModifiers: []planmodifier.String{
					planmodifiers.ImmutableAfterCreation{},
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username for ingest authentication. Must be a non-empty string. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for ingest authentication. Hashed server-side; never returned by the API. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					passwordComplexityValidator{},
				},
			},
			"algorithm": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Hashing algorithm for the password. One of `SHA256`, `SHA512_256`, `SHA512`, `MD5`. Changing this forces a new resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(models.HashAlgorithmSHA256),
						string(models.HashAlgorithmSHA512256),
						string(models.HashAlgorithmSHA512),
						string(models.HashAlgorithmMD5),
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional description for this credential.",
			},
			"expiry_date": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "RFC3339 expiry date for the credential. When omitted or null, the credential does not expire.",
			},
		},
	}
}

// --- Lifecycle ---

func (r *ingestCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ingestCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ingestCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := models.IngestCredentialCreateRequest{
		Username:  plan.Username.ValueString(),
		Password:  plan.Password.ValueString(),
		Algorithm: models.HashAlgorithm(plan.Algorithm.ValueString()),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		apiReq.Description = &v
	}

	cred, err := r.client.CreateIngestCredential(ctx, plan.StreamID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ingest credential", err.Error())
		return
	}

	// expiry_date cannot be set on create — apply it via a follow-up update if present.
	// The API may not immediately serve the new credential via its PUT endpoint, so we
	// retry on 404 with brief backoff to handle eventual-consistency propagation.
	if !plan.ExpiryDate.IsNull() && !plan.ExpiryDate.IsUnknown() {
		t, err := time.Parse(time.RFC3339, plan.ExpiryDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid expiry_date", fmt.Sprintf("expiry_date must be RFC3339: %s", err))
			return
		}
		updateReq := models.IngestCredentialUpdateRequest{ExpiryDate: &t}
		// Include description in the follow-up update so the response echoes it back
		// and the server does not clear it when the field is absent from the PUT body.
		if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
			v := plan.Description.ValueString()
			updateReq.Description = &v
		}

		const maxAttempts = 5
		delays := [maxAttempts]time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 0}
		var updateErr error
		for i := range maxAttempts {
			cred, updateErr = r.client.UpdateIngestCredential(ctx, plan.StreamID.ValueString(), cred.ID, updateReq)
			if updateErr == nil {
				break
			}
			if !client.IsNotFound(updateErr) || i == maxAttempts-1 {
				break
			}
			// 404 immediately after create — credential not yet propagated; wait and retry.
			select {
			case <-time.After(delays[i]):
			case <-ctx.Done():
				resp.Diagnostics.AddError("Error setting expiry_date on new ingest credential", ctx.Err().Error())
				return
			}
		}
		if updateErr != nil {
			resp.Diagnostics.AddError("Error setting expiry_date on new ingest credential", updateErr.Error())
			return
		}
	}

	// Password from plan is preserved — API never returns it
	resp.Diagnostics.Append(mapCredentialToState(cred, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ingestCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ingestCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cred, err := r.client.GetIngestCredentialByID(ctx, state.StreamID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ingest credential", err.Error())
		return
	}

	// Password is never returned by the API; preserve the current state value.
	resp.Diagnostics.Append(mapCredentialToState(cred, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ingestCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ingestCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := models.IngestCredentialUpdateRequest{}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		apiReq.Description = &v
	}
	if !plan.ExpiryDate.IsNull() && !plan.ExpiryDate.IsUnknown() {
		t, err := time.Parse(time.RFC3339, plan.ExpiryDate.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid expiry_date", fmt.Sprintf("expiry_date must be RFC3339: %s", err))
			return
		}
		apiReq.ExpiryDate = &t
	}

	cred, err := r.client.UpdateIngestCredential(ctx, plan.StreamID.ValueString(), plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ingest credential", err.Error())
		return
	}

	resp.Diagnostics.Append(mapCredentialToState(cred, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ingestCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ingestCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteIngestCredential(ctx, state.StreamID.ValueString(), state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting ingest credential", err.Error())
		return
	}
}

// ImportState accepts "stream_id/credential_id" as the composite import ID.
// Note: the password attribute will be unknown after import and must be set manually.
func (r *ingestCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected format: stream_id/credential_id, got: %q", req.ID),
		)
		return
	}
	streamID, credID := parts[0], parts[1]

	cred, err := r.client.GetIngestCredentialByID(ctx, streamID, credID)
	if err != nil {
		if client.IsNotFound(err) {
			resp.Diagnostics.AddError("Ingest credential not found", fmt.Sprintf("No credential %q found in stream %q", credID, streamID))
			return
		}
		resp.Diagnostics.AddError("Error importing ingest credential", err.Error())
		return
	}

	var state ingestCredentialResourceModel
	state.StreamID = types.StringValue(streamID)
	// Password cannot be recovered — leave as null; user must supply it in config.
	state.Password = types.StringNull()

	resp.Diagnostics.Append(mapCredentialToState(cred, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// --- Helpers ---

func mapCredentialToState(cred *models.IngestCredential, state *ingestCredentialResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	state.ID = types.StringValue(cred.ID)
	state.Username = types.StringValue(cred.Username)
	state.Algorithm = types.StringValue(string(cred.Algorithm))

	if cred.Description != nil {
		state.Description = types.StringValue(*cred.Description)
	} else {
		state.Description = types.StringNull()
	}
	if cred.ExpiryDate != nil {
		state.ExpiryDate = types.StringValue(cred.ExpiryDate.Format(time.RFC3339))
	} else {
		state.ExpiryDate = types.StringNull()
	}

	// Password is intentionally not set here — it is preserved from plan/state.
	// The API never returns the password (server-side hashed).

	return diags
}

// passwordComplexityValidator enforces the MSL5 password requirements:
// 4–50 characters, at least one uppercase, lowercase, digit, and special character.
type passwordComplexityValidator struct{}

func (v passwordComplexityValidator) Description(_ context.Context) string {
	return "Password must be 4–50 characters and contain at least one uppercase letter, lowercase letter, digit, and special character."
}

func (v passwordComplexityValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v passwordComplexityValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	pw := req.ConfigValue.ValueString()
	if len(pw) < 4 || len(pw) > 50 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid password length",
			fmt.Sprintf("Password must be between 4 and 50 characters, got %d.", len(pw)),
		)
		return
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, ch := range pw {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Password does not meet complexity requirements",
			"Password must contain at least one uppercase letter, one lowercase letter, one digit, and one special character.",
		)
	}
}
