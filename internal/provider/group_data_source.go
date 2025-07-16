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

type groupDataSource struct {
	client *client.Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &groupDataSource{}

// NewGroupDataSource returns a new instance of the group data source.
func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

// Metadata sets the data source type name for the TrueNAS group data source.
func (d *groupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the schema for the TrueNAS group data source.
func (d *groupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use this data source to get information about an specific group.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "This is the API identifier for the group.",
				Computed:            true,
			},
			"gid": schema.Int64Attribute{
				MarkdownDescription: "The Group ID (GID) is a unique number used to identify a Unix group.",
				Optional:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the group.",
				Optional:            true,
			},
			"sudo_commands": schema.ListAttribute{
				MarkdownDescription: "A list of commands that group members may execute with elevated privileges. " +
					"User is prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				MarkdownDescription: "A list of commands that group members may execute with elevated privileges. " +
					"User is not prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"smb": schema.BoolAttribute{
				MarkdownDescription: "If set to True, the group can be used for SMB share ACL entries. " +
					"The group is mapped to an NT group account on the TrueNAS SMB server and has a sid value.",
				Computed: true,
			},
			"userns_idmap": schema.DynamicAttribute{
				MarkdownDescription: "Specifies the subgid mapping for this group. " +
					"If DIRECT then the GID is be directly mapped to all containers. " +
					"Alternatively, the target GID may be explicitly specified. " +
					"If null, then the GID is not mapped.",
				Computed: true,
			},
			"users": schema.ListAttribute{
				MarkdownDescription: "A list a user IDs (UIDs) for local users who are members of this group.",
				ElementType:         types.Int64Type,
				Computed:            true,
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "Human-readable string to identify the group. Identical to the name key.",
				Computed:            true,
			},
			"builtin": schema.BoolAttribute{
				MarkdownDescription: "If True, the group is an internal system account for the TrueNAS server.",
				Computed:            true,
			},
			"local": schema.BoolAttribute{
				MarkdownDescription: "If True, the group is local to the TrueNAS server. " +
					"If False, the group is provided by a directory service.",
				Computed: true,
			},
			"sid": schema.StringAttribute{
				MarkdownDescription: "The Security Identifier (SID) of the user if the account an smb account. " +
					"The SMB server uses this value to check share access and for other purposes.",
				Computed: true,
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: "List of roles assigned to this groups. " +
					"Roles control administrative access to TrueNAS through the web UI and API.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

// Configure sets up the TrueNAS API client for the data source.
func (d *groupDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read fetches the group data and saves it into Terraform state.
func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform configuration data into the model
	var config TrueNASGroupModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TrueNASGroupModel
	service := NewTrueNASGroupService(ctx, d.client)
	resp.Diagnostics.Append(service.Query(config, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS group was successfully read from TrueNAS API")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Debug(ctx, "Successfully read TrueNAS group data source")
}
