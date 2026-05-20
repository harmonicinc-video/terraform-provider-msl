package stream

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// cidrStringValidator validates that a string is a valid CIDR notation (IPv4 or IPv6).
type cidrStringValidator struct{}

func (v cidrStringValidator) Description(_ context.Context) string {
	return "value must be a valid CIDR notation (e.g. 192.168.0.0/24 or 2001:db8::/32)"
}

func (v cidrStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()
	_, _, err := net.ParseCIDR(val)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR notation",
			"Expected a valid CIDR block (e.g. 192.168.0.0/24), got: "+val,
		)
	}
}
