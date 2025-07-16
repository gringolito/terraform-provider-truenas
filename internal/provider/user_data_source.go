package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type userDataSource struct {
	client *client.Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &groupDataSource{}

// NewUserDataSource returns a new instance of the user data source.
func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

// Metadata sets the data source type name for the TrueNAS user data source.
func (d *userDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the TrueNAS user data source.
func (d *userDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use this data source to get information about an specific user.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "This is the API identifier for the user.",
				Computed:            true,
			},
			"uid": schema.Int64Attribute{
				MarkdownDescription: "The User ID (UID) is a unique number used to identify a Unix user.",
				Optional:            true,
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "String used to uniquely identify the user on the server.",
				Optional:            true,
				Computed:            true,
			},
			"home": schema.StringAttribute{
				MarkdownDescription: "The local file system path for the user account's home directory.",
				Computed:            true,
			},
			"shell": schema.StringAttribute{
				MarkdownDescription: "The user local shell interpreter.",
				Computed:            true,
			},
			"full_name": schema.StringAttribute{
				MarkdownDescription: "Comment field to provide additional information about the user account. " +
					"Typically, this is the full name of the user or a short description of a service account.",
				Computed: true,
			},
			"smb": schema.BoolAttribute{
				MarkdownDescription: "The user account may be used to access SMB shares.",
				Computed:            true,
			},
			"userns_idmap": schema.DynamicAttribute{
				MarkdownDescription: "Specifies the subuid mapping for this user. " +
					"If DIRECT then the UID will be directly mapped to all containers. " +
					"Alternatively, the target UID may be explicitly specified. " +
					"If null, then the UID will not be mapped.",
				Computed: true,
			},
			"group": schema.Int64Attribute{
				MarkdownDescription: "The user's primary group Unix group GID.",
				Computed:            true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of additional groups to which the user belongs.",
				ElementType:         types.Int64Type,
				Computed:            true,
			},
			"password_disabled": schema.BoolAttribute{
				MarkdownDescription: "If set to true password authentication for the user account is disabled.",
				Computed:            true,
			},
			"ssh_password_enabled": schema.BoolAttribute{
				MarkdownDescription: "Allow the user to authenticate to the TrueNAS SSH server using a password.",
				Computed:            true,
			},
			"sshpubkey": schema.StringAttribute{
				MarkdownDescription: "SSH public keys corresponding to private keys that authenticate this user to the TrueNAS SSH server.",
				Computed:            true,
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "If set to true the account is locked. " +
					"The account cannot be used to authenticate to the TrueNAS server.",
				Computed: true,
			},
			"sudo_commands": schema.ListAttribute{
				MarkdownDescription: "A list of commands the user may execute with elevated privileges. " +
					"User is prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				MarkdownDescription: "A list of commands the user may execute with elevated privileges. " +
					"User is not prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"builtin": schema.BoolAttribute{
				MarkdownDescription: "If true, the user account is an internal system account for the TrueNAS server.",
				Computed:            true,
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the user.",
				Computed:            true,
			},
			"local": schema.BoolAttribute{
				MarkdownDescription: "If true, the account is local to the TrueNAS server. " +
					"If false, the account is provided by a directory service.",
				Computed: true,
			},
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the account is system-provided and most fields related to it may not be changed.",
				Computed:            true,
			},
			"twofactor_auth_configured": schema.BoolAttribute{
				MarkdownDescription: "If true, the account has been configured for two-factor authentication. " +
					"Users are prompted for a second factor when authenticating to the TrueNAS web UI and API. " +
					"They may also be prompted when signing in to the TrueNAS SSH server using a password (depending on global two-factor authentication settings).",
				Computed: true,
			},
			"sid": schema.StringAttribute{
				MarkdownDescription: "The Security Identifier (SID) of the user if the account an smb account. " +
					"The SMB server uses this value to check share access and for other purposes.",
				Computed: true,
			},
			"last_password_change": schema.DynamicAttribute{
				MarkdownDescription: "The date of the last password change for local user accounts.",
				Computed:            true,
			},
			"password_age": schema.Int64Attribute{
				MarkdownDescription: "The age in days of the password for local user accounts.",
				Computed:            true,
			},
			"password_change_required": schema.BoolAttribute{
				MarkdownDescription: "Password change for local user account is required on next login.",
				Computed:            true,
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: "List of roles assigned to this user's groups. " +
					"Roles control administrative access to TrueNAS through the web UI and API.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"api_keys": schema.ListAttribute{
				MarkdownDescription: "The IDs of the existing API keys for the user.",
				ElementType:         types.Int64Type,
				Computed:            true,
			},
		},
	}
}

// Configure sets up the TrueNAS API client for the data source.
func (d *userDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	d.client = api
}

// Read fetches the user data and saves it into Terraform state.
func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform configuration data into the model
	var config TrueNASUserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TrueNASUserModel
	service := NewTrueNASUserService(ctx, d.client)
	resp.Diagnostics.Append(service.Query(config, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS user was successfully read from TrueNAS API")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Debug(ctx, "Successfully read TrueNAS user data source")
}
