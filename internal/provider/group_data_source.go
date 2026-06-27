package provider

import (
	"context"
	"encoding/json"
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
				MarkdownDescription: "API identifier of the group. Set this or `name` (not both).",
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
				MarkdownDescription: "API identifiers of local users who are members of this group.",
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

	var groupID int64
	if hasID {
		groupID = config.Id.ValueInt64()
	} else {
		// Look up by name via group.query, then fetch the canonical object by ID.
		name := config.Name.ValueString()
		raw, err := d.caller.Call(ctx, "group.query",
			[]any{[]any{[]any{"name", "=", name}}})
		if err != nil {
			resp.Diagnostics.AddError("Error querying group by name", err.Error())
			return
		}
		var results []struct {
			Id int64 `json:"id"`
		}
		if err := json.Unmarshal(raw, &results); err != nil {
			resp.Diagnostics.AddError("Error parsing group query result", err.Error())
			return
		}
		if len(results) == 0 {
			resp.Diagnostics.AddError("Group not found",
				fmt.Sprintf("No group with name %q found.", name))
			return
		}
		groupID = results[0].Id
	}

	g, err := truenas.GroupGetInstance(ctx, d.caller, groupID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	var state GroupDataSourceModel
	groupToDataSourceModel(ctx, g, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func groupToDataSourceModel(ctx context.Context, g *truenas.Group, m *GroupDataSourceModel, diags *diag.Diagnostics) {
	rm := &GroupResourceModel{}
	groupToModel(ctx, g, rm, diags)
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
