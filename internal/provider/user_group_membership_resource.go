package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ resource.Resource = &UserGroupMembershipResource{}
var _ resource.ResourceWithImportState = &UserGroupMembershipResource{}

func NewUserGroupMembershipResource() resource.Resource {
	return &UserGroupMembershipResource{}
}

type UserGroupMembershipResource struct {
	caller client.Caller
}

type UserGroupMembershipResourceModel struct {
	Id       types.String `tfsdk:"id"`
	UserID   types.Int64  `tfsdk:"user_id"`
	GroupIDs types.Set    `tfsdk:"group_ids"`
}

func (r *UserGroupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_group_membership"
}

func (r *UserGroupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages additive supplementary group membership for a TrueNAS local user. " +
			"Uses read-modify-write semantics: only the declared groups are managed; groups assigned " +
			"outside this resource are preserved. Multiple `truenas_user_group_membership` blocks " +
			"for the same user compose additively.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to the Unix UID of the user as a string.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Unix UID of the user whose supplementary group membership is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"group_ids": schema.SetAttribute{
				Required:            true,
				ElementType:         types.Int64Type,
				MarkdownDescription: "Set of Unix GIDs to add as supplementary groups for the user.",
			},
		},
	}
}

func (r *UserGroupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserGroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserGroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := plan.UserID.ValueInt64()
	userAPIID, err := truenas.ResolveUserIDByUID(ctx, r.caller, uid)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving user by UID", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, userAPIID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredGIDs := setToInt64Slice(ctx, plan.GroupIDs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredAPIIDs, err := truenas.ResolveGroupIDsByGIDs(ctx, r.caller, declaredGIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving group API IDs", err.Error())
		return
	}

	u, err := truenas.UserGetInstance(ctx, r.caller, userAPIID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	merged := unionInt64(u.Groups, declaredAPIIDs)
	_, err = truenas.UserUpdate(ctx, r.caller, userAPIID, truenas.UserUpdateArgs{Groups: &merged})
	if err != nil {
		resp.Diagnostics.AddError("Error updating user group membership", err.Error())
		return
	}

	ugmReadIntoState(ctx, r.caller, userAPIID, uid, plan.GroupIDs, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserGroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserGroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userAPIID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, userAPIID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := state.UserID.ValueInt64()
	ugmReadIntoState(ctx, r.caller, userAPIID, uid, state.GroupIDs, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserGroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserGroupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state UserGroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userAPIID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	oldGIDs := setToInt64Slice(ctx, state.GroupIDs, &resp.Diagnostics)
	newGIDs := setToInt64Slice(ctx, plan.GroupIDs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	oldAPIIDs, err := truenas.ResolveGroupIDsByGIDs(ctx, r.caller, oldGIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving old group API IDs", err.Error())
		return
	}
	newAPIIDs, err := truenas.ResolveGroupIDsByGIDs(ctx, r.caller, newGIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving new group API IDs", err.Error())
		return
	}

	u, err := truenas.UserGetInstance(ctx, r.caller, userAPIID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	// result = (live − (old_declared − new_declared)) ∪ new_declared
	toRemove := differenceInt64(oldAPIIDs, newAPIIDs)
	base := differenceInt64(u.Groups, toRemove)
	result := unionInt64(base, newAPIIDs)

	_, err = truenas.UserUpdate(ctx, r.caller, userAPIID, truenas.UserUpdateArgs{Groups: &result})
	if err != nil {
		resp.Diagnostics.AddError("Error updating user group membership", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, userAPIID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := plan.UserID.ValueInt64()
	ugmReadIntoState(ctx, r.caller, userAPIID, uid, plan.GroupIDs, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserGroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserGroupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userAPIID, diags := readUserPrivateAPIID(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredGIDs := setToInt64Slice(ctx, state.GroupIDs, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	declaredAPIIDs, err := truenas.ResolveGroupIDsByGIDs(ctx, r.caller, declaredGIDs)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving group API IDs", err.Error())
		return
	}

	u, err := truenas.UserGetInstance(ctx, r.caller, userAPIID)
	if err != nil {
		if isNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	result := differenceInt64(u.Groups, declaredAPIIDs)
	_, err = truenas.UserUpdate(ctx, r.caller, userAPIID, truenas.UserUpdateArgs{Groups: &result})
	if err != nil {
		resp.Diagnostics.AddError("Error removing user group membership", err.Error())
	}
}

func (r *UserGroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected format <uid>:<gid1>,<gid2>, got %q", req.ID))
		return
	}

	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("UID must be an integer, got %q: %v", parts[0], err))
		return
	}

	gidStrs := strings.Split(parts[1], ",")
	gids := make([]int64, 0, len(gidStrs))
	for _, s := range gidStrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		gid, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			resp.Diagnostics.AddError("Invalid import ID",
				fmt.Sprintf("GID must be an integer, got %q: %v", s, err))
			return
		}
		gids = append(gids, gid)
	}
	if len(gids) == 0 {
		resp.Diagnostics.AddError("Invalid import ID", "At least one GID is required in the import address")
		return
	}

	userAPIID, err := truenas.ResolveUserIDByUID(ctx, r.caller, uid)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving user by UID", err.Error())
		return
	}

	resp.Diagnostics.Append(writeUserPrivateAPIID(ctx, resp.Private, userAPIID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupIDs := int64SliceToSet(ctx, gids, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.FormatInt(uid, 10))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), uid)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_ids"), groupIDs)...)
}

// ---- internal helpers -------------------------------------------------------

// ugmReadIntoState resolves the intersection of declared GIDs with live group membership
// and writes id, user_id, group_ids back into m.
func ugmReadIntoState(
	ctx context.Context,
	c client.Caller,
	userAPIID, uid int64,
	declaredGroupIDs types.Set,
	m *UserGroupMembershipResourceModel,
	diags *diag.Diagnostics,
) {
	u, err := truenas.UserGetInstance(ctx, c, userAPIID)
	if err != nil {
		diags.AddError("Error reading user", err.Error())
		return
	}

	declaredGIDs := setToInt64Slice(ctx, declaredGroupIDs, diags)
	if diags.HasError() {
		return
	}

	declaredAPIIDs, err := truenas.ResolveGroupIDsByGIDs(ctx, c, declaredGIDs)
	if err != nil {
		diags.AddError("Error resolving declared group API IDs", err.Error())
		return
	}

	// Detect deleted groups: if a declared GID no longer resolves, surface as error.
	if len(declaredAPIIDs) != len(declaredGIDs) {
		diags.AddError("Group not found",
			"One or more declared group GIDs no longer resolve to a group. The group may have been deleted.")
		return
	}

	intersectionAPIIDs := intersectInt64(declaredAPIIDs, u.Groups)

	intersectionGIDs, err := truenas.ResolveGroupGIDs(ctx, c, intersectionAPIIDs)
	if err != nil {
		diags.AddError("Error resolving group GIDs", err.Error())
		return
	}

	m.Id = types.StringValue(strconv.FormatInt(uid, 10))
	m.UserID = types.Int64Value(uid)
	m.GroupIDs = int64SliceToSet(ctx, intersectionGIDs, diags)
}

func setToInt64Slice(ctx context.Context, s types.Set, diags *diag.Diagnostics) []int64 {
	if s.IsNull() || s.IsUnknown() {
		return []int64{}
	}
	var out []int64
	diags.Append(s.ElementsAs(ctx, &out, false)...)
	if out == nil {
		return []int64{}
	}
	return out
}

// unionInt64 returns the union of a and b with no duplicates, preserving order.
func unionInt64(a, b []int64) []int64 {
	seen := make(map[int64]struct{}, len(a)+len(b))
	result := make([]int64, 0, len(a)+len(b))
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	for _, v := range b {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// differenceInt64 returns elements in a that are not in b.
func differenceInt64(a, b []int64) []int64 {
	remove := make(map[int64]struct{}, len(b))
	for _, v := range b {
		remove[v] = struct{}{}
	}
	result := make([]int64, 0, len(a))
	for _, v := range a {
		if _, ok := remove[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

// intersectInt64 returns elements present in both a and b.
func intersectInt64(a, b []int64) []int64 {
	set := make(map[int64]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	result := make([]int64, 0, len(a))
	for _, v := range a {
		if _, ok := set[v]; ok {
			result = append(result, v)
		}
	}
	return result
}
