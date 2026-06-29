package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}
var _ resource.ResourceWithConfigValidators = &UserResource{}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

type UserResource struct {
	caller client.Caller
}

type UserResourceModel struct {
	Id                      types.Int64  `tfsdk:"id"`
	Uid                     types.Int64  `tfsdk:"uid"`
	Username                types.String `tfsdk:"username"`
	FullName                types.String `tfsdk:"full_name"`
	Group                   types.Int64  `tfsdk:"group"`
	Groups                  types.Set    `tfsdk:"groups"`
	Password                types.String `tfsdk:"password"`
	PasswordWoVersion       types.Int64  `tfsdk:"password_wo_version"`
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

// privateStateKV matches the GetKey/SetKey methods on *privatestate.ProviderData
// without importing the internal package.
type privateStateKV interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func readUserPrivateAPIID(ctx context.Context, ps privateStateKV) (int64, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	raw, d := ps.GetKey(ctx, "user_api_id")
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return 0, diagnostics
	}
	if len(raw) == 0 {
		diagnostics.AddError("Missing private state", "Internal user API ID is not stored. Re-import the resource.")
		return 0, diagnostics
	}
	var v struct {
		APIID int64 `json:"api_id"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		diagnostics.AddError("Corrupt private state", fmt.Sprintf("Cannot decode internal user API ID: %v", err))
		return 0, diagnostics
	}
	return v.APIID, diagnostics
}

func writeUserPrivateAPIID(ctx context.Context, ps privateStateKV, apiID int64) diag.Diagnostics {
	raw, err := json.Marshal(struct {
		APIID int64 `json:"api_id"`
	}{APIID: apiID})
	if err != nil {
		var diagnostics diag.Diagnostics
		diagnostics.AddError("Cannot encode private state", err.Error())
		return diagnostics
	}
	return ps.SetKey(ctx, "user_api_id", raw)
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a TrueNAS local user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix UID of the user. Used as the import ID.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"uid": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Unix UID to assign. If omitted, TrueNAS assigns the next available UID.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Login name of the user.",
			},
			"full_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Full display name of the user.",
			},
			"group": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Unix GID of the primary group. Use `truenas_group.mygroup.id` to reference a managed group.",
			},
			"groups": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Unix GIDs of supplementary groups. Read-only — managed by `truenas_user_group_membership`.",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
				MarkdownDescription: "Account password. Write-only — never stored in state. Mutually exclusive with `password_disabled = true`. Bump `password_wo_version` to trigger a password update.",
			},
			"password_wo_version": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Monotonically increasing version counter. Increment this to trigger a password update when `password` changes. The provider sends the new password only when this value changes.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"password_disabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true`, the account has no password and cannot log in with a password. Mutually exclusive with `password`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"home": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Home directory path.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"home_create": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true`, TrueNAS creates the home directory if it does not exist.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"home_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Permissions of the home directory in octal notation (e.g. `\"700\"`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"shell": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Login shell path.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"smb": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true`, the user is added to the `builtin_users` SMB group and can authenticate to SMB shares.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"email": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Email address of the user. Null when not set.",
			},
			"ssh_password_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true`, password authentication over SSH is allowed for this user.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sshpubkey": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "SSH public key(s) authorised for this user.",
			},
			"locked": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true`, the account is locked and cannot log in.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sudo_commands": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands this user may execute with elevated privileges (password prompted).",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"sudo_commands_nopasswd": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands this user may execute with elevated privileges (no password required).",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"userns_idmap": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Subuid mapping for containers. `0` maps to `DIRECT`. A positive integer sets an explicit target UID. Omit for no mapping.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
				MarkdownDescription: "Security Identifier (SID) for SMB-enabled users. Null when SMB is disabled.",
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

// passwordMutualExclusionValidator errors when both password and password_disabled=true are set.
type passwordMutualExclusionValidator struct{}

func (v *passwordMutualExclusionValidator) Description(_ context.Context) string {
	return "`password` and `password_disabled = true` are mutually exclusive"
}

func (v *passwordMutualExclusionValidator) MarkdownDescription(_ context.Context) string {
	return "`password` and `password_disabled = true` are mutually exclusive"
}

func (v *passwordMutualExclusionValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	passwordSet := !config.Password.IsNull() && !config.Password.IsUnknown()
	disabledTrue := !config.PasswordDisabled.IsNull() && !config.PasswordDisabled.IsUnknown() && config.PasswordDisabled.ValueBool()
	if passwordSet && disabledTrue {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Conflicting attributes",
			"`password` cannot be set when `password_disabled = true`.",
		)
	}
}

type smbPasswordDisabledValidator struct{}

func (v *smbPasswordDisabledValidator) Description(_ context.Context) string {
	return "`smb = true` and `password_disabled = true` are mutually exclusive"
}

func (v *smbPasswordDisabledValidator) MarkdownDescription(_ context.Context) string {
	return "`smb = true` and `password_disabled = true` are mutually exclusive"
}

func (v *smbPasswordDisabledValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	smbTrue := !config.Smb.IsNull() && !config.Smb.IsUnknown() && config.Smb.ValueBool()
	disabledTrue := !config.PasswordDisabled.IsNull() && !config.PasswordDisabled.IsUnknown() && config.PasswordDisabled.ValueBool()
	if smbTrue && disabledTrue {
		resp.Diagnostics.AddAttributeError(
			path.Root("smb"),
			"Conflicting attributes",
			"`smb = true` cannot be used with `password_disabled = true`. TrueNAS requires SMB authentication to be disabled when password login is disabled.",
		)
	}
}

func (r *UserResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		&passwordMutualExclusionValidator{},
		&smbPasswordDisabledValidator{},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Caller)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected client.Caller, got %T", req.ProviderData))
		return
	}
	r.caller = c
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// WriteOnly attributes are nullified in the plan — read them from the config.
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupAPIID, err := truenas.ResolveGroupIDByGID(ctx, r.caller, plan.Group.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error resolving primary group", err.Error())
		return
	}

	sudoCmds := setToStringSlice(ctx, plan.SudoCommands, &resp.Diagnostics)
	sudoCmdsNP := setToStringSlice(ctx, plan.SudoCommandsNopasswd, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	args := truenas.UserCreateArgs{
		Username:             plan.Username.ValueString(),
		FullName:             plan.FullName.ValueString(),
		Group:                &groupAPIID,
		SudoCommands:         sudoCmds,
		SudoCommandsNopasswd: sudoCmdsNP,
		UsernsIdmap:          usernsIdmapToJSON(plan.UsernsIdmap),
	}
	if !plan.Uid.IsNull() && !plan.Uid.IsUnknown() {
		v := plan.Uid.ValueInt64()
		args.Uid = &v
	}
	if !plan.Home.IsNull() && !plan.Home.IsUnknown() {
		args.Home = plan.Home.ValueString()
	}
	if !plan.Shell.IsNull() && !plan.Shell.IsUnknown() {
		args.Shell = plan.Shell.ValueString()
	}
	if !plan.Smb.IsNull() && !plan.Smb.IsUnknown() {
		v := plan.Smb.ValueBool()
		args.Smb = &v
	} else if !plan.PasswordDisabled.IsNull() && plan.PasswordDisabled.ValueBool() {
		// TrueNAS rejects password_disabled=true unless smb=false is sent explicitly
		// (smb defaults to true on the API). Validator ensures smb≠true here.
		smbFalse := false
		args.Smb = &smbFalse
	}
	if !plan.PasswordDisabled.IsNull() && !plan.PasswordDisabled.IsUnknown() {
		v := plan.PasswordDisabled.ValueBool()
		args.PasswordDisabled = &v
	}
	if !plan.SshPasswordEnabled.IsNull() && !plan.SshPasswordEnabled.IsUnknown() {
		v := plan.SshPasswordEnabled.ValueBool()
		args.SshPasswordEnabled = &v
	}
	if !plan.Locked.IsNull() && !plan.Locked.IsUnknown() {
		v := plan.Locked.ValueBool()
		args.Locked = &v
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		v := plan.Email.ValueString()
		args.Email = &v
	}
	if !plan.Sshpubkey.IsNull() && !plan.Sshpubkey.IsUnknown() {
		v := plan.Sshpubkey.ValueString()
		args.Sshpubkey = &v
	}
	if !plan.HomeCreate.IsNull() && !plan.HomeCreate.IsUnknown() {
		v := plan.HomeCreate.ValueBool()
		args.HomeCreate = &v
	}
	if !plan.HomeMode.IsNull() && !plan.HomeMode.IsUnknown() {
		args.HomeMode = plan.HomeMode.ValueString()
	}
	// WriteOnly: password is nullified in the plan; read it from config.
	if !config.Password.IsNull() && !config.Password.IsUnknown() {
		if plan.PasswordDisabled.IsNull() || !plan.PasswordDisabled.ValueBool() {
			v := config.Password.ValueString()
			args.Password = &v
		}
	}

	u, err := truenas.UserCreateSafe(ctx, r.caller, args)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, u.Id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fresh, err := truenas.UserGetInstanceSafe(ctx, r.caller, u.Id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user after create", err.Error())
		return
	}

	userToModel(ctx, r.caller, fresh, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	u, err := truenas.UserGetInstanceSafe(ctx, r.caller, apiID)
	if err != nil {
		if isNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, u.Id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userToModel(ctx, r.caller, u, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// WriteOnly attributes are nullified in the plan — read them from the config.
	var config UserResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupAPIID, err := truenas.ResolveGroupIDByGID(ctx, r.caller, plan.Group.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error resolving primary group", err.Error())
		return
	}

	smb := plan.Smb.ValueBool()
	passwordDisabled := plan.PasswordDisabled.ValueBool()
	sshPasswordEnabled := plan.SshPasswordEnabled.ValueBool()
	locked := plan.Locked.ValueBool()
	homeCreate := plan.HomeCreate.ValueBool()

	sudoCmds := setToStringSlice(ctx, plan.SudoCommands, &resp.Diagnostics)
	sudoCmdsNP := setToStringSlice(ctx, plan.SudoCommandsNopasswd, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	args := truenas.UserUpdateArgs{
		Username:             plan.Username.ValueString(),
		FullName:             plan.FullName.ValueString(),
		Group:                &groupAPIID,
		Smb:                  &smb,
		PasswordDisabled:     &passwordDisabled,
		SshPasswordEnabled:   &sshPasswordEnabled,
		Locked:               &locked,
		HomeCreate:           &homeCreate,
		SudoCommands:         &sudoCmds,
		SudoCommandsNopasswd: &sudoCmdsNP,
		// Groups nil → omitted from JSON → supplementary memberships preserved.
		Groups: nil,
	}

	if !plan.Home.IsNull() && !plan.Home.IsUnknown() {
		args.Home = plan.Home.ValueString()
	}
	if !plan.Shell.IsNull() && !plan.Shell.IsUnknown() {
		args.Shell = plan.Shell.ValueString()
	}
	if !plan.HomeMode.IsNull() && !plan.HomeMode.IsUnknown() {
		args.HomeMode = plan.HomeMode.ValueString()
	}
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		v := plan.Email.ValueString()
		args.Email = &v
	}
	if !plan.Sshpubkey.IsNull() && !plan.Sshpubkey.IsUnknown() {
		v := plan.Sshpubkey.ValueString()
		args.Sshpubkey = &v
	}
	args.UsernsIdmap = usernsIdmapToJSON(plan.UsernsIdmap)

	// WriteOnly: password is nullified in the plan; read it from config.
	// Send password only when version counter changed and account is not disabled.
	if !plan.PasswordWoVersion.Equal(state.PasswordWoVersion) && !plan.PasswordDisabled.ValueBool() {
		if !config.Password.IsNull() && !config.Password.IsUnknown() {
			v := config.Password.ValueString()
			args.Password = &v
		}
	}

	_, err = truenas.UserUpdateSafe(ctx, r.caller, apiID, args)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, apiID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fresh, err := truenas.UserGetInstanceSafe(ctx, r.caller, apiID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user after update", err.Error())
		return
	}

	userToModel(ctx, r.caller, fresh, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	apiID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := truenas.UserDelete(ctx, r.caller, apiID); err != nil {
		if isNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	uid, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected a Unix UID integer, got %q: %v", req.ID, err))
		return
	}

	raw, err := truenas.UserQuery(ctx, r.caller, truenas.QueryFilter{Field: "uid", Op: "=", Value: uid})
	if err != nil {
		resp.Diagnostics.AddError("Error querying user by UID", err.Error())
		return
	}
	var users []struct {
		Id  int64 `json:"id"`
		Uid int64 `json:"uid"`
	}
	if err := json.Unmarshal(raw, &users); err != nil {
		resp.Diagnostics.AddError("Error parsing user query result", err.Error())
		return
	}
	if len(users) == 0 {
		resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user with UID %d", uid))
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, users[0].Id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(uid))
}

// ---- conversion helpers ----------------------------------------------------

func primaryGroupGID(u *truenas.User) (int64, error) {
	raw, ok := u.Group["bsdgrp_gid"]
	if !ok {
		return 0, fmt.Errorf("group response missing bsdgrp_gid field")
	}
	var gid int64
	if err := json.Unmarshal(raw, &gid); err != nil {
		return 0, fmt.Errorf("parsing bsdgrp_gid: %w", err)
	}
	return gid, nil
}

func userToModel(ctx context.Context, c client.Caller, u *truenas.User, m *UserResourceModel, diags *diag.Diagnostics) {
	var username string
	if err := json.Unmarshal(u.Username, &username); err != nil {
		diags.AddError("Error decoding username", fmt.Sprintf("Cannot decode username JSON: %v", err))
		return
	}

	gid, err := primaryGroupGID(u)
	if err != nil {
		diags.AddError("Error reading primary group", err.Error())
		return
	}

	gids, err := truenas.ResolveGroupGIDs(ctx, c, u.Groups)
	if err != nil {
		diags.AddError("Error resolving supplementary group GIDs", err.Error())
		return
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	m.Id = types.Int64Value(u.Uid)
	m.Uid = types.Int64Value(u.Uid)
	m.Username = types.StringValue(username)
	m.FullName = types.StringValue(u.FullName)
	m.Group = types.Int64Value(gid)
	m.Groups = int64SliceToSet(ctx, gids, diags)
	m.PasswordDisabled = types.BoolValue(u.PasswordDisabled)
	m.Home = types.StringValue(u.Home)
	m.Shell = types.StringValue(u.Shell)
	m.Smb = types.BoolValue(u.Smb)
	m.SshPasswordEnabled = types.BoolValue(u.SshPasswordEnabled)
	m.Locked = types.BoolValue(u.Locked)
	m.SudoCommands = stringSliceToSet(ctx, u.SudoCommands, diags)
	m.SudoCommandsNopasswd = stringSliceToSet(ctx, u.SudoCommandsNopasswd, diags)
	m.UsernsIdmap = usernsIdmapFromJSON(u.UsernsIdmap)
	m.Builtin = types.BoolValue(u.Builtin)
	m.Immutable = types.BoolValue(u.Immutable)
	m.Local = types.BoolValue(u.Local)
	m.TwofactorAuthConfigured = types.BoolValue(u.TwofactorAuthConfigured)
	m.Roles = stringSliceToSet(ctx, u.Roles, diags)

	if u.Sid != nil {
		m.Sid = types.StringValue(*u.Sid)
	} else {
		m.Sid = types.StringNull()
	}
	if u.Email != nil {
		m.Email = types.StringValue(*u.Email)
	} else {
		m.Email = types.StringNull()
	}
	if u.Sshpubkey != nil {
		m.Sshpubkey = types.StringValue(*u.Sshpubkey)
	} else {
		m.Sshpubkey = types.StringNull()
	}

	// home_create, home_mode, password, and password_wo_version are write-only
	// API signals not returned by get_instance. Preserve existing plan/state
	// values; only resolve Unknown → Null so Terraform doesn't error with
	// "unknown value after apply" on the first create.
	if m.HomeCreate.IsUnknown() {
		m.HomeCreate = types.BoolNull()
	}
	if m.HomeMode.IsUnknown() {
		m.HomeMode = types.StringNull()
	}
	if m.PasswordWoVersion.IsUnknown() {
		m.PasswordWoVersion = types.Int64Null()
	}
}
