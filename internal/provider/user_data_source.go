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

var _ datasource.DataSource = &UserDataSource{}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

type UserDataSource struct {
	caller client.Caller
}

type UserDataSourceModel struct {
	Id                      types.Int64  `tfsdk:"id"`
	Username                types.String `tfsdk:"username"`
	FullName                types.String `tfsdk:"full_name"`
	Group                   types.Int64  `tfsdk:"group"`
	Groups                  types.Set    `tfsdk:"groups"`
	PasswordDisabled        types.Bool   `tfsdk:"password_disabled"`
	Home                    types.String `tfsdk:"home"`
	HomeCreate              types.Bool   `tfsdk:"home_create"`
	HomeMode                types.String `tfsdk:"home_mode"`
	Shell                   types.String `tfsdk:"shell"`
	Smb                     types.Bool   `tfsdk:"smb"`
	Email                   types.String `tfsdk:"email"`
	SshPasswordEnabled      types.Bool   `tfsdk:"ssh_password_enabled"`
	Sshpubkey               types.String `tfsdk:"sshpubkey"`
	Locked                  types.Bool   `tfsdk:"locked"`
	SudoCommands            types.Set    `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd    types.Set    `tfsdk:"sudo_commands_nopasswd"`
	UsernsIdmap             types.Int64  `tfsdk:"userns_idmap"`
	Builtin                 types.Bool   `tfsdk:"builtin"`
	Immutable               types.Bool   `tfsdk:"immutable"`
	Local                   types.Bool   `tfsdk:"local"`
	Sid                     types.String `tfsdk:"sid"`
	Roles                   types.Set    `tfsdk:"roles"`
	TwofactorAuthConfigured types.Bool   `tfsdk:"twofactor_auth_configured"`
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a TrueNAS user. Look up by `id` (Unix UID) or `username` (exactly one must be set).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unix UID of the user. Set this or `username` (not both).",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Login name of the user. Set this or `id` (not both).",
			},
			"full_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full display name of the user.",
			},
			"group": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix GID of the primary group.",
			},
			"groups": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Unix GIDs of supplementary groups.",
			},
			"password_disabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the account has no password.",
			},
			"home": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Home directory path.",
			},
			"home_create": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether TrueNAS creates the home directory if missing.",
			},
			"home_mode": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Permissions of the home directory in octal notation.",
			},
			"shell": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Login shell path.",
			},
			"smb": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the user can authenticate to SMB shares.",
			},
			"email": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Email address of the user.",
			},
			"ssh_password_enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether password authentication over SSH is allowed.",
			},
			"sshpubkey": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "SSH public key(s) authorised for this user.",
			},
			"locked": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the account is locked.",
			},
			"sudo_commands": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands this user may execute with elevated privileges (password prompted).",
			},
			"sudo_commands_nopasswd": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands this user may execute with elevated privileges (no password).",
			},
			"userns_idmap": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Subuid mapping for containers. `0` means `DIRECT`. Null means no mapping.",
			},
			"builtin": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this is an internal system user.",
			},
			"immutable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this user entry can be modified.",
			},
			"local": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether this user is local to the TrueNAS server.",
			},
			"sid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Security Identifier (SID) for SMB-enabled users.",
			},
			"roles": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "TrueNAS roles assigned to this user.",
			},
			"twofactor_auth_configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether two-factor authentication is configured for this user.",
			},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config UserDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.Id.IsNull() && !config.Id.IsUnknown()
	hasUsername := !config.Username.IsNull() && !config.Username.IsUnknown()

	if hasID == hasUsername {
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Exactly one of `id` or `username` must be set.",
		)
		return
	}

	var u *truenas.User
	var err error

	if hasID {
		raw, qErr := truenas.UserQuery(ctx, d.caller, truenas.QueryFilter{Field: "uid", Op: "=", Value: config.Id.ValueInt64()})
		if qErr != nil {
			resp.Diagnostics.AddError("Error querying user", qErr.Error())
			return
		}
		var users []struct {
			Id int64 `json:"id"`
		}
		if jsonErr := json.Unmarshal(raw, &users); jsonErr != nil {
			resp.Diagnostics.AddError("Error parsing user query", jsonErr.Error())
			return
		}
		if len(users) == 0 {
			resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user with UID %d", config.Id.ValueInt64()))
			return
		}
		u, err = truenas.UserGetInstanceSafe(ctx, d.caller, users[0].Id)
	} else {
		u, err = truenas.UserGetByUsername(ctx, d.caller, config.Username.ValueString())
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	var state UserDataSourceModel
	userToDataSourceModel(ctx, d.caller, u, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func userToDataSourceModel(ctx context.Context, c client.Caller, u *truenas.User, m *UserDataSourceModel, diags *diag.Diagnostics) {
	rm := &UserResourceModel{}
	userToModel(ctx, c, u, rm, diags)
	if diags.HasError() {
		return
	}
	m.Id = rm.Id
	m.Username = rm.Username
	m.FullName = rm.FullName
	m.Group = rm.Group
	m.Groups = rm.Groups
	m.PasswordDisabled = rm.PasswordDisabled
	m.Home = rm.Home
	m.HomeCreate = rm.HomeCreate
	m.HomeMode = rm.HomeMode
	m.Shell = rm.Shell
	m.Smb = rm.Smb
	m.Email = rm.Email
	m.SshPasswordEnabled = rm.SshPasswordEnabled
	m.Sshpubkey = rm.Sshpubkey
	m.Locked = rm.Locked
	m.SudoCommands = rm.SudoCommands
	m.SudoCommandsNopasswd = rm.SudoCommandsNopasswd
	m.UsernsIdmap = rm.UsernsIdmap
	m.Builtin = rm.Builtin
	m.Immutable = rm.Immutable
	m.Local = rm.Local
	m.Sid = rm.Sid
	m.Roles = rm.Roles
	m.TwofactorAuthConfigured = rm.TwofactorAuthConfigured
}
