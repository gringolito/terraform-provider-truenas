package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure TrueNASProvider satisfies various provider interfaces.
var _ provider.Provider = &TrueNASProvider{}
var _ provider.ProviderWithFunctions = &TrueNASProvider{}
var _ provider.ProviderWithEphemeralResources = &TrueNASProvider{}

// TrueNASProvider defines the provider implementation.
// Implements the Terraform provider interface for TrueNAS.
type TrueNASProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// TrueNASProviderModel describes the provider configuration data model.
type TrueNASProviderModel struct {
	Server    types.String `tfsdk:"server"`
	APIKey    types.String `tfsdk:"api_key"`
	APITLS    types.Bool   `tfsdk:"api_tls"`
	VerifySSL types.Bool   `tfsdk:"verify_ssl"`
}

// Metadata sets the provider type name and version for Terraform.
func (p *TrueNASProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "truenas"
	resp.Version = p.version
}

// Schema returns the provider schema, describing available configuration attributes.
func (p *TrueNASProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"server": schema.StringAttribute{
				MarkdownDescription: "TrueNAS server address. " +
					"Alternatively, can be configured using the `TRUENAS_SERVER` environment variable.",
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "TrueNAS API key. " +
					"Alternatively, can be configured using the `TRUENAS_API_KEY` environment variable.",
				Sensitive: true,
				Optional:  true,
			},
			"api_tls": schema.BoolAttribute{
				MarkdownDescription: "Enable Websocket TLS encryption. " +
					"Alternatively, can be configured using the `TRUENAS_API_TLS` environment variable.",
				Optional: true,
			},
			"verify_ssl": schema.BoolAttribute{
				MarkdownDescription: "Verify TLS certificates (useful for self-signed certificates). " +
					"Alternatively, can be configured using the `TRUENAS_VERIFY_SSL` environment variable.",
				Optional: true,
			},
		},
	}
}

// Configure configures the provider using configuration and environment variables.
// Reads configuration from Terraform and environment, validates, and sets up client.
func (p *TrueNASProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config TrueNASProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	// If practitioner provided a configuration value for any of the attributes, it must be a known value.
	if config.Server.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("server"),
			"Unknown TrueNAS Server Address",
			"The provider cannot create the TrueNAS client as there is an unknown configuration value for the "+
				"TrueNAS server address. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use the TRUENAS_SERVER environment variable.",
		)
	}
	if config.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown TrueNAS API Key",
			"The provider cannot create the TrueNAS client as there is an unknown configuration value for the "+
				"TrueNAS API Key. Either target apply the source of the value first, set the value "+
				"statically in the configuration, or use the TRUENAS_API_KEY environment variable.",
		)
	}

	// Configuration values should take precedence over environment variable values, if found.
	server := os.Getenv("TRUENAS_SERVER")
	if !config.Server.IsNull() {
		server = config.Server.ValueString()
	}

	apiKey := os.Getenv("TRUENAS_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}

	verifySSL, err := strconv.ParseBool(os.Getenv("TRUENAS_VERIFY_SSL"))
	if err != nil {
		verifySSL = true
	}
	if !config.VerifySSL.IsNull() {
		verifySSL = config.VerifySSL.ValueBool()
	}

	apiTLS, err := strconv.ParseBool(os.Getenv("TRUENAS_API_TLS"))
	if err != nil {
		apiTLS = true
	}
	if !config.APITLS.IsNull() {
		apiTLS = config.APITLS.ValueBool()
	}

	if server == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("server"),
			"Missing TrueNAS Server Address Configuration",
			"While configuring the provider, the endpoint was not found in the TRUENAS_SERVER environment variable "+
				" or provider configuration block server attribute.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing TrueNAS API Key Configuration",
			"While configuring the provider, the API Key was not found in the TRUENAS_API_KEY environment variable "+
				"or provider configuration block api_key attribute.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, fmt.Sprintf("Connecting to TrueNAS server at %q", server))

	apiProtocol := "ws://"
	if apiTLS {
		apiProtocol = "wss://"
	}

	api, err := client.NewClient(server, apiProtocol, verifySSL, apiKey)
	if err != nil {
		if e, ok := err.(client.ClientError); ok {
			resp.Diagnostics.AddError("TrueNAS Client Failed", e.Error())
			return
		}
		if e, ok := err.(client.AuthenticationError); ok {
			resp.Diagnostics.AddError("TrueNAS Authentication Failed", e.Error())
			return
		}
		resp.Diagnostics.AddError("TrueNAS API Error", err.Error())
	}

	tflog.Debug(ctx, "TrueNAS login successful!")

	resp.DataSourceData = api
	resp.ResourceData = api
}

// Resources returns the list of supported Terraform resources.
func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewUserResource,
		NewGroupResource,
		NewDatasetResource,
		NewDatasetPermissionResource,
		NewShareNFSResource,
	}
}

// EphemeralResources returns the list of supported ephemeral resources.
func (p *TrueNASProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

// DataSources returns the list of supported Terraform data sources.
func (p *TrueNASProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewUserDataSource,
		NewGroupDataSource,
		NewDatasetDataSource,
		NewShareNFSDataSource,
	}
}

// Functions returns the list of supported provider functions.
func (p *TrueNASProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

// New returns a new instance of the TrueNASProvider with the given version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TrueNASProvider{
			version: version,
		}
	}
}
