package provider

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	goversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

var _ provider.Provider = &TrueNASProvider{}

type TrueNASProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type TrueNASProviderModel struct {
	Host               types.String `tfsdk:"host"`
	APIKey             types.String `tfsdk:"api_key"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func (p *TrueNASProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "truenas"
	resp.Version = p.version
}

func (p *TrueNASProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provider for managing TrueNAS SCALE resources via the TrueNAS API.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "TrueNAS hostname or IP address. Can also be set with the `TRUENAS_HOST` environment variable.",
			},
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API key for authentication. Cannot be used together with `username` and `password`. Can also be set with the `TRUENAS_API_KEY` environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for authentication. Must be used together with `password`. Cannot be used together with `api_key`. Can also be set with the `TRUENAS_USERNAME` environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for authentication. Must be used together with `username`. Cannot be used together with `api_key`. Can also be set with the `TRUENAS_PASSWORD` environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Skip TLS certificate verification. Can also be set with the `TRUENAS_INSECURE_SKIP_VERIFY` environment variable.",
			},
		},
	}
}

func (p *TrueNASProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var model TrueNASProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := stringFromModelOrEnv(model.Host, "TRUENAS_HOST")
	apiKey := stringFromModelOrEnv(model.APIKey, "TRUENAS_API_KEY")
	username := stringFromModelOrEnv(model.Username, "TRUENAS_USERNAME")
	password := stringFromModelOrEnv(model.Password, "TRUENAS_PASSWORD")

	insecureSkipVerify := false
	if !model.InsecureSkipVerify.IsNull() && !model.InsecureSkipVerify.IsUnknown() {
		insecureSkipVerify = model.InsecureSkipVerify.ValueBool()
	} else if envVal := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY"); envVal != "" {
		parsed, err := strconv.ParseBool(envVal)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid TRUENAS_INSECURE_SKIP_VERIFY",
				"TRUENAS_INSECURE_SKIP_VERIFY must be a valid boolean value: "+err.Error(),
			)
			return
		}
		insecureSkipVerify = parsed
	}

	if req.TerraformVersion != "" {
		tfver, err := goversion.NewVersion(req.TerraformVersion)
		if err == nil {
			minver, _ := goversion.NewVersion("1.11")
			if tfver.LessThan(minver) {
				resp.Diagnostics.AddError(
					"Unsupported Terraform Version",
					"This provider requires Terraform 1.11 or later. Got: "+req.TerraformVersion,
				)
				return
			}
		}
	}

	if host == "" {
		resp.Diagnostics.AddError(
			"Missing host",
			"The `host` attribute or `TRUENAS_HOST` environment variable must be set.",
		)
		return
	}

	resp.Diagnostics.Append(validateCredentials(apiKey, username, password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	caller, err := client.NewWebSocketClient(host, apiKey, username, password, insecureSkipVerify)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to connect to TrueNAS",
			err.Error(),
		)
		return
	}

	checkServerVersion(ctx, caller, resp)

	var c client.Caller = caller
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDatasetResource,
		NewGroupResource,
		NewUserResource,
		NewUserGroupMembershipResource,
	}
}

func (p *TrueNASProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDatasetDataSource,
		NewGroupDataSource,
		NewPoolDataSource,
		NewUserDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TrueNASProvider{
			version: version,
		}
	}
}

func validateCredentials(apiKey, username, password string) diag.Diagnostics {
	var diags diag.Diagnostics

	hasAPIKey := apiKey != ""
	hasUsername := username != ""
	hasPassword := password != ""

	if hasAPIKey && (hasUsername || hasPassword) {
		diags.AddError(
			"Conflicting credential sets",
			"Cannot use `api_key` together with `username` or `password`. Use either `api_key` alone, or `username` and `password` together.",
		)
		return diags
	}

	if hasUsername && !hasPassword {
		diags.AddError(
			"Incomplete credential set",
			"`username` requires `password` to also be set.",
		)
		return diags
	}

	if !hasUsername && hasPassword {
		diags.AddError(
			"Incomplete credential set",
			"`password` requires `username` to also be set.",
		)
		return diags
	}

	if !hasAPIKey && !hasUsername && !hasPassword {
		diags.AddError(
			"Missing credentials",
			"Must provide either `api_key` or both `username` and `password`.",
		)
		return diags
	}

	return diags
}

func stringFromModelOrEnv(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}

func checkServerVersion(ctx context.Context, caller client.Caller, resp *provider.ConfigureResponse) {
	raw, err := caller.Call(ctx, "system.version", nil)
	if err != nil {
		resp.Diagnostics.AddWarning("Unable to check TrueNAS version", err.Error())
		return
	}

	var ver string
	if err := json.Unmarshal(raw, &ver); err != nil {
		resp.Diagnostics.AddWarning("Unable to parse TrueNAS version", err.Error())
		return
	}

	if !strings.HasPrefix(ver, "TrueNAS-25.10.") {
		resp.Diagnostics.AddWarning(
			"Unsupported TrueNAS version",
			"This provider is validated against TrueNAS 25.10. Detected version: "+ver+". Other versions may work but are untested.",
		)
	}
}
