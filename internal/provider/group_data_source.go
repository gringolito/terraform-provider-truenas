package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ datasource.DataSource = &GroupDataSource{}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

type GroupDataSource struct {
	caller client.Caller
}

type GroupDataSourceModel struct {
	Id                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Gid                  types.Int64  `tfsdk:"gid"`
	Smb                  types.Bool   `tfsdk:"smb"`
	SudoCommands         types.Set    `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.Set    `tfsdk:"sudo_commands_nopasswd"`
	UsernsIdmap          types.Int64  `tfsdk:"userns_idmap"`
	Builtin              types.Bool   `tfsdk:"builtin"`
	Immutable            types.Bool   `tfsdk:"immutable"`
	Local                types.Bool   `tfsdk:"local"`
	Sid                  types.String `tfsdk:"sid"`
	Roles                types.Set    `tfsdk:"roles"`
	Users                types.Set    `tfsdk:"users"`
}

func (d *GroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a TrueNAS group. Look up by `id` or `name` (exactly one must be set).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Identifier of the group. Set this or `name` (not both).",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Name of the group. Set this or `id` (not both).",
			},
			"gid": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Group ID (GID).",
			},
			"smb": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the group can be used for SMB share ACL entries.",
			},
			"sudo_commands": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands group members may execute with elevated privileges (password prompted).",
			},
			"sudo_commands_nopasswd": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands group members may execute with elevated privileges (no password).",
			},
			"userns_idmap": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Subgid mapping for containers. `0` means `DIRECT`. Null means no mapping.",
			},
			"builtin": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is an internal system group.",
			},
			"immutable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group entry can be modified.",
			},
			"local": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this group is local to the TrueNAS server.",
			},
			"sid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Security Identifier (SID) for SMB-enabled groups.",
			},
			"roles": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "TrueNAS roles assigned to this group.",
			},
			"users": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Unix UIDs of local users who are members of this group.",
			},
		},
	}
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Caller)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected client.Caller, got %T", req.ProviderData))
		return
	}
	d.caller = c
}

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.Id.IsNull() && !config.Id.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown()

	if hasID == hasName {
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Exactly one of `id` or `name` must be set.",
		)
		return
	}

	var g *truenas.Group
	var err error
	if hasID {
		g, err = truenas.GroupGetInstance(ctx, d.caller, config.Id.ValueInt64())
	} else {
		g, err = truenas.GroupGetByName(ctx, d.caller, config.Name.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	var state GroupDataSourceModel
	groupToDataSourceModel(ctx, d.caller, g, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func groupToDataSourceModel(ctx context.Context, c client.Caller, g *truenas.Group, m *GroupDataSourceModel, diags *diag.Diagnostics) {
	rm := &GroupResourceModel{}
	groupToModelWithUIDs(ctx, c, g, rm, diags)
	if diags.HasError() {
		return
	}
	m.Id = rm.Id
	m.Name = rm.Name
	m.Gid = rm.Gid
	m.Smb = rm.Smb
	m.SudoCommands = rm.SudoCommands
	m.SudoCommandsNopasswd = rm.SudoCommandsNopasswd
	m.UsernsIdmap = rm.UsernsIdmap
	m.Builtin = rm.Builtin
	m.Immutable = rm.Immutable
	m.Local = rm.Local
	m.Sid = rm.Sid
	m.Roles = rm.Roles
	m.Users = rm.Users
}
