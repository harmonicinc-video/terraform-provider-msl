// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---- defaultInt64 ----

func TestDefaultInt64_UsesValueWhenSet(t *testing.T) {
	got := defaultInt64(types.Int64Value(10), 99)
	if got != 10 {
		t.Errorf("defaultInt64 = %d, want 10", got)
	}
}

func TestDefaultInt64_UsesFallbackWhenNull(t *testing.T) {
	got := defaultInt64(types.Int64Null(), 99)
	if got != 99 {
		t.Errorf("defaultInt64 = %d, want 99 (null)", got)
	}
}

func TestDefaultInt64_UsesFallbackWhenZero(t *testing.T) {
	got := defaultInt64(types.Int64Value(0), 5)
	if got != 5 {
		t.Errorf("defaultInt64 = %d, want 5 (zero)", got)
	}
}

func TestDefaultInt64_UsesFallbackWhenUnknown(t *testing.T) {
	got := defaultInt64(types.Int64Unknown(), 7)
	if got != 7 {
		t.Errorf("defaultInt64 = %d, want 7 (unknown)", got)
	}
}

// ---- attrPath ----

func TestAttrPath_ReturnsRootPath(t *testing.T) {
	p := attrPath("endpoint")
	if p.String() != `Path{}` && p.String() == "" {
		// path.Root returns a path; just confirm it doesn't panic and returns something
		t.Errorf("attrPath returned empty path")
	}
}

// ---- Metadata ----

func TestProvider_Metadata(t *testing.T) {
	p := &mslProvider{version: "2.3.4"}
	var resp provider.MetadataResponse
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)

	if resp.TypeName != "msl" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "msl")
	}
	if resp.Version != "2.3.4" {
		t.Errorf("Version = %q, want %q", resp.Version, "2.3.4")
	}
}

// ---- Schema ----

func TestProvider_Schema_RequiredAttributes(t *testing.T) {
	p := &mslProvider{}
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes
	for _, required := range []string{"endpoint", "api_token"} {
		attr, ok := attrs[required]
		if !ok {
			t.Errorf("required attribute %q not found in provider schema", required)
			continue
		}
		if !attr.IsRequired() {
			t.Errorf("attribute %q should be Required", required)
		}
	}
}

func TestProvider_Schema_OptionalAttributes(t *testing.T) {
	p := &mslProvider{}
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	attrs := resp.Schema.Attributes
	for _, optional := range []string{"request_timeout", "max_retries", "debug_logging"} {
		if _, ok := attrs[optional]; !ok {
			t.Errorf("optional attribute %q not found in provider schema", optional)
		}
	}
}

func TestProvider_Schema_APITokenIsSensitive(t *testing.T) {
	p := &mslProvider{}
	var resp provider.SchemaResponse
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

	attr, ok := resp.Schema.Attributes["api_token"]
	if !ok {
		t.Fatal("api_token attribute not found")
	}
	if !attr.IsSensitive() {
		t.Error("api_token should be marked Sensitive")
	}
}

// ---- Resources / DataSources ----

func TestProvider_Resources_Count(t *testing.T) {
	p := &mslProvider{}
	resources := p.Resources(context.Background())
	if len(resources) != 4 {
		t.Errorf("Resources() len = %d, want 4", len(resources))
	}
}

func TestProvider_DataSources_Count(t *testing.T) {
	p := &mslProvider{}
	dataSources := p.DataSources(context.Background())
	if len(dataSources) != 4 {
		t.Errorf("DataSources() len = %d, want 4", len(dataSources))
	}
}

func TestProvider_Functions_Empty(t *testing.T) {
	p := &mslProvider{}
	fns := p.Functions(context.Background())
	if len(fns) != 0 {
		t.Errorf("Functions() len = %d, want 0", len(fns))
	}
}
