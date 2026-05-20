package ingestcredential

import (
	"context"
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- mapCredentialToState ----

func TestMapCredentialToState_BasicFields(t *testing.T) {
	cred := &models.IngestCredential{
		ID:        "cred-1",
		Username:  "user1",
		Algorithm: models.HashAlgorithmSHA256,
	}

	var state ingestCredentialResourceModel
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}

	if state.ID.ValueString() != "cred-1" {
		t.Errorf("ID = %q, want %q", state.ID.ValueString(), "cred-1")
	}
	if state.Username.ValueString() != "user1" {
		t.Errorf("Username = %q, want %q", state.Username.ValueString(), "user1")
	}
	if state.Algorithm.ValueString() != "SHA256" {
		t.Errorf("Algorithm = %q, want %q", state.Algorithm.ValueString(), "SHA256")
	}
}

func TestMapCredentialToState_DescriptionSet(t *testing.T) {
	desc := "my credential"
	cred := &models.IngestCredential{
		ID:          "cred-1",
		Username:    "user1",
		Algorithm:   models.HashAlgorithmMD5,
		Description: &desc,
	}

	var state ingestCredentialResourceModel
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}

	if state.Description.IsNull() {
		t.Fatal("Description should not be null")
	}
	if state.Description.ValueString() != "my credential" {
		t.Errorf("Description = %q, want %q", state.Description.ValueString(), "my credential")
	}
}

func TestMapCredentialToState_DescriptionNil(t *testing.T) {
	cred := &models.IngestCredential{
		ID:        "cred-1",
		Username:  "user1",
		Algorithm: models.HashAlgorithmSHA512,
	}

	var state ingestCredentialResourceModel
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}

	if !state.Description.IsNull() {
		t.Errorf("Description should be null, got %q", state.Description.ValueString())
	}
}

func TestMapCredentialToState_ExpiryDateSet(t *testing.T) {
	expiry := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	cred := &models.IngestCredential{
		ID:         "cred-1",
		Username:   "user1",
		Algorithm:  models.HashAlgorithmSHA256,
		ExpiryDate: &expiry,
	}

	var state ingestCredentialResourceModel
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}

	if state.ExpiryDate.IsNull() {
		t.Fatal("ExpiryDate should not be null")
	}
	if state.ExpiryDate.ValueString() != expiry.Format(time.RFC3339) {
		t.Errorf("ExpiryDate = %q, want %q", state.ExpiryDate.ValueString(), expiry.Format(time.RFC3339))
	}
}

func TestMapCredentialToState_ExpiryDateNil(t *testing.T) {
	cred := &models.IngestCredential{
		ID:        "cred-1",
		Username:  "user1",
		Algorithm: models.HashAlgorithmSHA256,
	}

	var state ingestCredentialResourceModel
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}

	if !state.ExpiryDate.IsNull() {
		t.Errorf("ExpiryDate should be null, got %q", state.ExpiryDate.ValueString())
	}
}

func TestMapCredentialToState_PasswordNotOverwritten(t *testing.T) {
	// The API never returns the password; mapCredentialToState must not touch it.
	cred := &models.IngestCredential{
		ID:        "cred-1",
		Username:  "user1",
		Algorithm: models.HashAlgorithmSHA256,
	}
	state := ingestCredentialResourceModel{
		Password: types.StringValue("original-password"),
	}
	diags := mapCredentialToState(cred, &state)
	if diags.HasError() {
		t.Fatalf("mapCredentialToState() returned errors: %v", diags)
	}
	if state.Password.ValueString() != "original-password" {
		t.Errorf("Password should be preserved, got %q", state.Password.ValueString())
	}
}

// ---- passwordComplexityValidator ----

func validatePassword(t *testing.T, pw string) (hasErrors bool) {
	t.Helper()
	v := passwordComplexityValidator{}
	req := validator.StringRequest{
		Path:        path.Root("password"),
		ConfigValue: types.StringValue(pw),
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	return resp.Diagnostics.HasError()
}

func TestPasswordValidator_Valid(t *testing.T) {
	passwords := []string{
		"Pass1!",
		"Abc1@def",
		"My$ecure1Password",
	}
	for _, pw := range passwords {
		if validatePassword(t, pw) {
			t.Errorf("expected %q to pass validation, but got errors", pw)
		}
	}
}

func TestPasswordValidator_TooShort(t *testing.T) {
	if !validatePassword(t, "A1!") {
		t.Error("expected error for password shorter than 4 characters")
	}
}

func TestPasswordValidator_TooLong(t *testing.T) {
	pw := "Aa1!" + string(make([]byte, 47)) // 51 chars total
	if !validatePassword(t, pw) {
		t.Error("expected error for password longer than 50 characters")
	}
}

func TestPasswordValidator_MissingUppercase(t *testing.T) {
	if !validatePassword(t, "pass1!ab") {
		t.Error("expected error when no uppercase letter")
	}
}

func TestPasswordValidator_MissingLowercase(t *testing.T) {
	if !validatePassword(t, "PASS1!AB") {
		t.Error("expected error when no lowercase letter")
	}
}

func TestPasswordValidator_MissingDigit(t *testing.T) {
	if !validatePassword(t, "Password!") {
		t.Error("expected error when no digit")
	}
}

func TestPasswordValidator_MissingSpecialChar(t *testing.T) {
	if !validatePassword(t, "Password1") {
		t.Error("expected error when no special character")
	}
}

func TestPasswordValidator_NullValue_NoError(t *testing.T) {
	v := passwordComplexityValidator{}
	req := validator.StringRequest{
		Path:        path.Root("password"),
		ConfigValue: types.StringNull(),
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Error("expected no error for null password value")
	}
}

func TestPasswordValidator_Description(t *testing.T) {
	v := passwordComplexityValidator{}
	if v.Description(context.Background()) == "" {
		t.Error("Description() should not be empty")
	}
}

// ---- username LengthAtLeast(1) validator ----

func validateUsername(t *testing.T, username string) (hasErrors bool) {
	t.Helper()
	v := stringvalidator.LengthAtLeast(1)
	req := validator.StringRequest{
		Path:        path.Root("username"),
		ConfigValue: types.StringValue(username),
	}
	var resp validator.StringResponse
	v.ValidateString(context.Background(), req, &resp)
	return resp.Diagnostics.HasError()
}

func TestUsernameValidator_Valid(t *testing.T) {
	usernames := []string{"alice", "bob123", "encoder-1"}
	for _, u := range usernames {
		if validateUsername(t, u) {
			t.Errorf("expected %q to pass username validation, but got errors", u)
		}
	}
}

func TestUsernameValidator_EmptyString_Rejected(t *testing.T) {
	if !validateUsername(t, "") {
		t.Error("expected error for empty username, but validation passed")
	}
}

// ---- schema: immutable fields ----

func TestIngestCredentialSchema_StreamID_IsImmutable(t *testing.T) {
	r := &ingestCredentialResource{}
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	rawAttr, ok := resp.Schema.Attributes["stream_id"]
	if !ok {
		t.Fatal("stream_id not found in schema")
	}
	strAttr, ok := rawAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("stream_id is not a schema.StringAttribute")
	}
	if !strAttr.Required {
		t.Error("stream_id should be Required")
	}
	if len(strAttr.PlanModifiers) == 0 {
		t.Error("stream_id should have ImmutableAfterCreation plan modifier")
	}
}
