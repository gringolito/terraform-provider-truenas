package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ resource.Resource = &GroupResource{}
var _ resource.ResourceWithImportState = &GroupResource{}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

type GroupResource struct {
	caller client.Caller
}

type GroupResourceModel struct {
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

func (r *GroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a TrueNAS local group.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Unix GID of the group.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the group.",
			},
			"gid": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Group ID (GID). If omitted, TrueNAS assigns the next available GID.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"smb": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "When `true` the group can be used for SMB share ACL entries.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sudo_commands": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands that group members may execute with elevated privileges (password prompted).",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"sudo_commands_nopasswd": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Commands that group members may execute with elevated privileges (no password required).",
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"userns_idmap": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Subgid mapping for containers. `0` maps to `DIRECT` (the GID is mapped directly). A positive integer sets an explicit target GID. Omit for no mapping.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
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
				MarkdownDescription: "Security Identifier (SID) for SMB-enabled groups. Null when SMB is disabled.",
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

func readGroupPrivateAPIID(ctx context.Context, ps privateStateKV) (int64, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	raw, d := ps.GetKey(ctx, "group_api_id")
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return 0, diagnostics
	}
	if len(raw) == 0 {
		diagnostics.AddError("Missing private state", "Internal group API ID is not stored. Re-import the resource.")
		return 0, diagnostics
	}
	var v struct {
		APIID int64 `json:"api_id"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		diagnostics.AddError("Corrupt private state", fmt.Sprintf("Cannot decode internal group API ID: %v", err))
		return 0, diagnostics
	}
	return v.APIID, diagnostics
}

func writeGroupPrivateAPIID(ctx context.Context, ps privateStateKV, apiID int64) diag.Diagnostics {
	raw, err := json.Marshal(struct {
		APIID int64 `json:"api_id"`
	}{APIID: apiID})
	if err != nil {
		var diagnostics diag.Diagnostics
		diagnostics.AddError("Cannot encode private state", err.Error())
		return diagnostics
	}
	return ps.SetKey(ctx, "group_api_id", raw)
}

func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	args := truenas.GroupCreateArgs{
		Name:                 plan.Name.ValueString(),
		SudoCommands:         setToStringSlice(ctx, plan.SudoCommands, &resp.Diagnostics),
		SudoCommandsNopasswd: setToStringSlice(ctx, plan.SudoCommandsNopasswd, &resp.Diagnostics),
		UsernsIdmap:          usernsIdmapToJSON(plan.UsernsIdmap),
	}
	if !plan.Gid.IsNull() && !plan.Gid.IsUnknown() {
		v := plan.Gid.ValueInt64()
		args.Gid = &v
	}
	if !plan.Smb.IsNull() && !plan.Smb.IsUnknown() {
		v := plan.Smb.ValueBool()
		args.Smb = &v
	}
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := truenas.GroupCreate(ctx, r.caller, args)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", err.Error())
		return
	}

	resp.Diagnostics.Append(writeGroupPrivateAPIID(ctx, resp.Private, id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := truenas.GroupGetInstance(ctx, r.caller, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group after create", err.Error())
		return
	}
	groupToModelWithUIDs(ctx, r.caller, g, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiID, diags := readGroupPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := truenas.GroupGetInstance(ctx, r.caller, apiID)
	if err != nil {
		if isNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	resp.Diagnostics.Append(writeGroupPrivateAPIID(ctx, resp.Private, g.Id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupToModelWithUIDs(ctx, r.caller, g, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state GroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	smb := plan.Smb.ValueBool()
	sudoCmds := setToStringSlice(ctx, plan.SudoCommands, &resp.Diagnostics)
	sudoCmdsNP := setToStringSlice(ctx, plan.SudoCommandsNopasswd, &resp.Diagnostics)
	args := truenas.GroupUpdateArgs{
		Name:                 plan.Name.ValueString(),
		Smb:                  &smb,
		SudoCommands:         &sudoCmds,
		SudoCommandsNopasswd: &sudoCmdsNP,
		// Users is a read-only computed attribute — omit (nil) to preserve membership.
		Users: nil,
	}
	// Only send userns_idmap when the plan has an explicit value; omit it
	// (leave nil so json omitempty kicks in) when unchanged/null.
	if !plan.UsernsIdmap.IsNull() && !plan.UsernsIdmap.IsUnknown() {
		args.UsernsIdmap = usernsIdmapToJSON(plan.UsernsIdmap)
	}
	apiID, diags := readGroupPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := truenas.GroupUpdate(ctx, r.caller, apiID, args); err != nil {
		resp.Diagnostics.AddError("Error updating group", err.Error())
		return
	}

	g, err := truenas.GroupGetInstance(ctx, r.caller, apiID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading group after update", err.Error())
		return
	}

	resp.Diagnostics.Append(writeGroupPrivateAPIID(ctx, resp.Private, apiID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupToModelWithUIDs(ctx, r.caller, g, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	apiID, diags := readGroupPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := truenas.GroupDelete(ctx, r.caller, apiID); err != nil {
		if isNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting group", err.Error())
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	gid, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected a Unix GID integer, got %q: %v", req.ID, err))
		return
	}

	apiID, err := truenas.ResolveGroupIDByGID(ctx, r.caller, gid)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving group by GID", err.Error())
		return
	}

	resp.Diagnostics.Append(writeGroupPrivateAPIID(ctx, resp.Private, apiID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.State.SetAttribute(ctx, path.Root("id"), types.Int64Value(gid))
}

// ---- conversion helpers ----------------------------------------------------

// usernsIdmapToJSON converts the Terraform Int64 (nullable) to the API's
// polymorphic userns_idmap value:
//
//	null TF value  →  JSON null
//	0              →  JSON string "DIRECT"
//	n > 0          →  JSON integer n
func usernsIdmapToJSON(v types.Int64) json.RawMessage {
	if v.IsNull() || v.IsUnknown() {
		return json.RawMessage("null")
	}
	n := v.ValueInt64()
	if n == 0 {
		return json.RawMessage(`"DIRECT"`)
	}
	b, _ := json.Marshal(n)
	return b
}

// usernsIdmapFromJSON parses the API's polymorphic userns_idmap into TF Int64:
//
//	JSON null    →  types.Int64Null()
//	"DIRECT"     →  types.Int64Value(0)
//	integer n    →  types.Int64Value(n)
func usernsIdmapFromJSON(raw json.RawMessage) types.Int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return types.Int64Null()
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "DIRECT" {
			return types.Int64Value(0)
		}
		return types.Int64Null()
	}
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return types.Int64Value(n)
	}
	return types.Int64Null()
}

func setToStringSlice(ctx context.Context, s types.Set, diags *diag.Diagnostics) []string {
	if s.IsNull() || s.IsUnknown() {
		return []string{}
	}
	var out []string
	diags.Append(s.ElementsAs(ctx, &out, false)...)
	if out == nil {
		return []string{}
	}
	return out
}

func stringSliceToSet(ctx context.Context, vals []string, diags *diag.Diagnostics) types.Set {
	if vals == nil {
		vals = []string{}
	}
	s, d := types.SetValueFrom(ctx, types.StringType, vals)
	diags.Append(d...)
	return s
}

func int64SliceToSet(ctx context.Context, vals []int64, diags *diag.Diagnostics) types.Set {
	if vals == nil {
		vals = []int64{}
	}
	s, d := types.SetValueFrom(ctx, types.Int64Type, vals)
	diags.Append(d...)
	return s
}

func isNotFoundErr(err error) bool {
	var apiErr *client.APIError
	return errors.As(err, &apiErr) && apiErr.IsNotFound()
}

func groupToModel(ctx context.Context, g *truenas.Group, m *GroupResourceModel, diags *diag.Diagnostics) {
	m.Id = types.Int64Value(g.Gid)
	m.Name = types.StringValue(g.Name)
	m.Gid = types.Int64Value(g.Gid)
	m.Smb = types.BoolValue(g.Smb)
	m.SudoCommands = stringSliceToSet(ctx, g.SudoCommands, diags)
	m.SudoCommandsNopasswd = stringSliceToSet(ctx, g.SudoCommandsNopasswd, diags)
	m.UsernsIdmap = usernsIdmapFromJSON(g.UsernsIdmap)
	m.Builtin = types.BoolValue(g.Builtin)
	m.Immutable = types.BoolValue(g.Immutable)
	m.Local = types.BoolValue(g.Local)
	if g.Sid != nil {
		m.Sid = types.StringValue(*g.Sid)
	} else {
		m.Sid = types.StringNull()
	}
	m.Roles = stringSliceToSet(ctx, g.Roles, diags)
	m.Users = int64SliceToSet(ctx, g.Users, diags)
}

// groupToModelWithUIDs is like groupToModel but resolves the group's user
// internal IDs to Unix UIDs via a batched user.query call.
func groupToModelWithUIDs(ctx context.Context, c client.Caller, g *truenas.Group, m *GroupResourceModel, diags *diag.Diagnostics) {
	groupToModel(ctx, g, m, diags)
	if diags.HasError() {
		return
	}
	uids, err := truenas.ResolveUserUIDs(ctx, c, g.Users)
	if err != nil {
		diags.AddError("Error resolving user UIDs", err.Error())
		return
	}
	m.Users = int64SliceToSet(ctx, uids, diags)
}
