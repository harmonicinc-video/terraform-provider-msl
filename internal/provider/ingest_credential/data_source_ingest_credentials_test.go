package ingestcredential

import (
	"testing"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// credAttrString reads a string attribute from a types.Object.
func credAttrString(obj types.Object, key string) string {
	v, ok := obj.Attributes()[key]
	if !ok {
		return ""
	}
	sv, ok := v.(types.String)
	if !ok {
		return ""
	}
	return sv.ValueString()
}

// ---- credentialToObject ----

func TestCredentialToObject_BasicFields(t *testing.T) {
	desc := "my cred"
	expiry := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	c := models.IngestCredential{
		ID:          "cred-1",
		Username:    "encoder1",
		Algorithm:   models.HashAlgorithmSHA256,
		Description: &desc,
		ExpiryDate:  &expiry,
	}

	obj, diags := credentialToObject(c, "stream-abc")
	if diags.HasError() {
		t.Fatalf("credentialToObject() returned errors: %v", diags)
	}

	cases := map[string]string{
		"id":          "cred-1",
		"stream_id":   "stream-abc",
		"username":    "encoder1",
		"algorithm":   "SHA256",
		"description": "my cred",
		"expiry_date": expiry.Format("2006-01-02T15:04:05Z07:00"),
	}
	for key, want := range cases {
		if got := credAttrString(obj, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestCredentialToObject_NilDescriptionProducesEmptyString(t *testing.T) {
	c := models.IngestCredential{
		ID:          "cred-1",
		Username:    "encoder1",
		Algorithm:   models.HashAlgorithmSHA512,
		Description: nil,
	}
	obj, diags := credentialToObject(c, "s-1")
	if diags.HasError() {
		t.Fatalf("credentialToObject() returned errors: %v", diags)
	}
	if got := credAttrString(obj, "description"); got != "" {
		t.Errorf("description should be empty string for nil, got %q", got)
	}
}

func TestCredentialToObject_NilExpiryDateProducesEmptyString(t *testing.T) {
	c := models.IngestCredential{
		ID:        "cred-1",
		Username:  "encoder1",
		Algorithm: models.HashAlgorithmSHA256,
	}
	obj, diags := credentialToObject(c, "s-1")
	if diags.HasError() {
		t.Fatalf("credentialToObject() returned errors: %v", diags)
	}
	if got := credAttrString(obj, "expiry_date"); got != "" {
		t.Errorf("expiry_date should be empty string for nil, got %q", got)
	}
}

func TestCredentialToObject_StreamIDPassedThrough(t *testing.T) {
	c := models.IngestCredential{ID: "cred-1", Algorithm: models.HashAlgorithmMD5}
	obj, diags := credentialToObject(c, "my-stream-id")
	if diags.HasError() {
		t.Fatalf("credentialToObject() returned errors: %v", diags)
	}
	if got := credAttrString(obj, "stream_id"); got != "my-stream-id" {
		t.Errorf("stream_id = %q, want %q", got, "my-stream-id")
	}
}

func TestCredentialToObject_AlgorithmValues(t *testing.T) {
	algos := []models.HashAlgorithm{
		models.HashAlgorithmSHA256,
		models.HashAlgorithmSHA512256,
		models.HashAlgorithmSHA512,
		models.HashAlgorithmMD5,
	}
	for _, algo := range algos {
		c := models.IngestCredential{ID: "c", Algorithm: algo}
		obj, diags := credentialToObject(c, "s")
		if diags.HasError() {
			t.Fatalf("credentialToObject() for algo %q returned errors: %v", algo, diags)
		}
		if got := credAttrString(obj, "algorithm"); got != string(algo) {
			t.Errorf("algorithm = %q, want %q", got, string(algo))
		}
	}
}

func TestCredentialToObject_ContainsAllExpectedKeys(t *testing.T) {
	c := models.IngestCredential{ID: "cred-1", Algorithm: models.HashAlgorithmSHA256}
	obj, diags := credentialToObject(c, "s-1")
	if diags.HasError() {
		t.Fatalf("credentialToObject() returned errors: %v", diags)
	}
	for key := range credentialAttrTypes {
		if _, ok := obj.Attributes()[key]; !ok {
			t.Errorf("missing attribute %q in object", key)
		}
	}
}
