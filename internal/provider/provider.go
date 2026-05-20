package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/harmonicinc-video/terraform-provider-msl/internal/client"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/event"
	ingestcredential "github.com/harmonicinc-video/terraform-provider-msl/internal/provider/ingest_credential"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/origin"
	"github.com/harmonicinc-video/terraform-provider-msl/internal/provider/stream"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure mslProvider implements the provider.Provider interface.
var _ provider.Provider = &mslProvider{}
var _ provider.ProviderWithFunctions = &mslProvider{}

// mslProvider is the concrete implementation of the MSL5 Terraform provider.
type mslProvider struct {
	version string
}

// mslProviderModel holds the schema-decoded configuration values.
type mslProviderModel struct {
	Endpoint       types.String `tfsdk:"endpoint"`
	APIToken       types.String `tfsdk:"api_token"`
	RequestTimeout types.Int64  `tfsdk:"request_timeout"`
	MaxRetries     types.Int64  `tfsdk:"max_retries"`
	DebugLogging   types.Bool   `tfsdk:"debug_logging"`
}

// New returns a provider.Provider factory function for the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &mslProvider{version: version}
	}
}

func (p *mslProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "msl"
	resp.Version = p.version
}

func (p *mslProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The **MSL5 Live** provider manages Akamai Media Services Live (MSL5) resources via the MSL5 REST API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL of the MSL5 API, e.g. `https://api.msl.example.com`.",
			},
			"api_token": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Bearer token for MSL5 API authentication.",
			},
			"request_timeout": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "HTTP request timeout in seconds. Defaults to `30`.",
			},
			"max_retries": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of retries on transient errors. Defaults to `3`.",
			},
			"debug_logging": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable verbose debug logging. Never logs sensitive values. Defaults to `false`.",
			},
		},
	}
}

func (p *mslProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg mslProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.Endpoint.IsUnknown() || cfg.APIToken.IsUnknown() {
		resp.Diagnostics.AddError(
			"Unknown provider configuration",
			"The provider cannot be configured with unknown values. Ensure endpoint and api_token are known at plan time.",
		)
		return
	}

	endpoint := cfg.Endpoint.ValueString()
	apiToken := cfg.APIToken.ValueString()

	if endpoint == "" {
		resp.Diagnostics.AddAttributeError(
			attrPath("endpoint"),
			"Missing MSL5 API endpoint",
			"Set the 'endpoint' provider attribute.",
		)
	}
	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			attrPath("api_token"),
			"Missing MSL5 API token",
			"Set the 'api_token' provider attribute.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	timeout := defaultInt64(cfg.RequestTimeout, 30)
	maxRetries := defaultInt64(cfg.MaxRetries, 3)
	debugLog := !cfg.DebugLogging.IsNull() && cfg.DebugLogging.ValueBool()

	c, err := client.New(client.Config{
		BaseURL:        endpoint,
		Token:          apiToken,
		RequestTimeout: time.Duration(timeout) * time.Second,
		MaxRetries:     int(maxRetries),
		DebugLog:       debugLog,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure MSL5 client", fmt.Sprintf("Error: %s", err))
		return
	}

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *mslProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		origin.NewOriginResource,
		stream.NewStreamResource,
		event.NewEventResource,
		ingestcredential.NewIngestCredentialResource,
	}
}

func (p *mslProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		origin.NewOriginsDataSource,
		stream.NewStreamsDataSource,
		event.NewEventsDataSource,
		ingestcredential.NewIngestCredentialsDataSource,
	}
}

func (p *mslProvider) Functions(_ context.Context) []func() function.Function {
	return nil
}

// defaultInt64 returns val if known and non-zero, otherwise fallback.
func defaultInt64(val types.Int64, fallback int64) int64 {
	if !val.IsNull() && !val.IsUnknown() && val.ValueInt64() != 0 {
		return val.ValueInt64()
	}
	return fallback
}
