package stream

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCidrStringValidator_ValidIPv4CIDR(t *testing.T) {
	for _, cidr := range []string{"192.168.0.0/24", "10.0.0.0/8", "0.0.0.0/0", "203.0.113.5/32"} {
		t.Run(cidr, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("allowed_ips"),
				ConfigValue: types.StringValue(cidr),
			}
			var resp validator.StringResponse
			cidrStringValidator{}.ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %v", cidr, resp.Diagnostics)
			}
		})
	}
}

func TestCidrStringValidator_ValidIPv6CIDR(t *testing.T) {
	for _, cidr := range []string{"2001:db8::/32", "::/0", "fe80::/10"} {
		t.Run(cidr, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("allowed_ips"),
				ConfigValue: types.StringValue(cidr),
			}
			var resp validator.StringResponse
			cidrStringValidator{}.ValidateString(context.Background(), req, &resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("expected no error for %q, got: %v", cidr, resp.Diagnostics)
			}
		})
	}
}

func TestCidrStringValidator_InvalidCIDR(t *testing.T) {
	for _, bad := range []string{
		"192.168.0.1",       // plain IP, no prefix length
		"192.168.0.0/33",    // prefix length out of range for IPv4
		"not-a-cidr",        // garbage
		"",                  // empty string
		"256.0.0.0/8",       // invalid octet
		"192.168.1.0/24/16", // double slash
	} {
		t.Run(bad, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("allowed_ips"),
				ConfigValue: types.StringValue(bad),
			}
			var resp validator.StringResponse
			cidrStringValidator{}.ValidateString(context.Background(), req, &resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error for %q, got none", bad)
			}
		})
	}
}

func TestCidrStringValidator_NullIsSkipped(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("allowed_ips"),
		ConfigValue: types.StringNull(),
	}
	var resp validator.StringResponse
	cidrStringValidator{}.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for null value, got: %v", resp.Diagnostics)
	}
}

func TestCidrStringValidator_UnknownIsSkipped(t *testing.T) {
	req := validator.StringRequest{
		Path:        path.Root("allowed_ips"),
		ConfigValue: types.StringUnknown(),
	}
	var resp validator.StringResponse
	cidrStringValidator{}.ValidateString(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("expected no error for unknown value, got: %v", resp.Diagnostics)
	}
}

func TestCidrStringValidator_Descriptions(t *testing.T) {
	v := cidrStringValidator{}
	if v.Description(context.Background()) == "" {
		t.Error("Description() should not be empty")
	}
	if v.MarkdownDescription(context.Background()) == "" {
		t.Error("MarkdownDescription() should not be empty")
	}
}
