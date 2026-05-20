package provider

import "github.com/hashicorp/terraform-plugin-framework/path"

// attrPath creates a single-step attribute path (convenience helper).
func attrPath(attr string) path.Path {
	return path.Root(attr)
}
