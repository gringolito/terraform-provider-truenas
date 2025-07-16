package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TrueNASUserModel describes the schema and state for a TrueNAS user.
type TrueNASUserModel struct {
	ID                      types.String  `tfsdk:"id"`
	UserID                  types.Int64   `tfsdk:"uid"`
	Username                types.String  `tfsdk:"username"`
	Home                    types.String  `tfsdk:"home"`
	Shell                   types.String  `tfsdk:"shell"`
	FullName                types.String  `tfsdk:"full_name"`
	Smb                     types.Bool    `tfsdk:"smb"`
	UsernsIdmap             types.Dynamic `tfsdk:"userns_idmap"`
	GroupID                 types.Int64   `tfsdk:"group"`
	Groups                  types.List    `tfsdk:"groups"`
	PasswordDisabled        types.Bool    `tfsdk:"password_disabled"`
	SSHPasswordEnabled      types.Bool    `tfsdk:"ssh_password_enabled"`
	Sshpubkey               types.String  `tfsdk:"sshpubkey"`
	Locked                  types.Bool    `tfsdk:"locked"`
	SudoCommands            types.List    `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd    types.List    `tfsdk:"sudo_commands_nopasswd"`
	Email                   types.String  `tfsdk:"email"`
	Builtin                 types.Bool    `tfsdk:"builtin"`
	Local                   types.Bool    `tfsdk:"local"`
	Immutable               types.Bool    `tfsdk:"immutable"`
	TwofactorAuthConfigured types.Bool    `tfsdk:"twofactor_auth_configured"`
	Sid                     types.String  `tfsdk:"sid"`
	LastPasswordChange      types.Dynamic `tfsdk:"last_password_change"`
	PasswordAge             types.Int64   `tfsdk:"password_age"`
	PasswordChangeRequired  types.Bool    `tfsdk:"password_change_required"`
	Roles                   types.List    `tfsdk:"roles"`
	APIKeys                 types.List    `tfsdk:"api_keys"`
}

func (u *TrueNASUserModel) GetID(id *int64) diag.Diagnostics {
	var diags diag.Diagnostics
	parsedID, err := strconv.ParseInt(u.ID.ValueString(), 10, 0)
	if err != nil {
		diags.AddError("Failed to get TrueNAS User ID", err.Error())
	}

	*id = parsedID
	return diags
}

// TrueNASUserResourceModel describes the schema and state for a TrueNAS user resource.
type TrueNASUserResourceModel struct {
	TrueNASUserModel
	GroupCreate    types.Bool   `tfsdk:"group_create"`
	HomeCreate     types.Bool   `tfsdk:"home_create"`
	HomeMode       types.String `tfsdk:"home_mode"`
	RandomPassword types.Bool   `tfsdk:"random_password"`
	Password       types.String `tfsdk:"password"`
}

func (u *TrueNASUserResourceModel) CopyFromPlan(plan TrueNASUserResourceModel) {
	u.CopyFromState(plan)
}

func (u *TrueNASUserResourceModel) CopyFromState(state TrueNASUserResourceModel) {
	u.GroupCreate = state.GroupCreate
	u.HomeCreate = state.HomeCreate
	u.HomeMode = state.HomeMode
	u.RandomPassword = state.RandomPassword
}

type trueNASUserService struct {
	trueNASService
}

func NewTrueNASUserService(ctx context.Context, client *client.Client) trueNASUserService {
	return trueNASUserService{trueNASService{ctx, client, diag.Diagnostics{}}}
}

func (s *trueNASUserService) modelFromAPI(user client.UserQueryResultItem) TrueNASUserModel {
	var model TrueNASUserModel

	model.ID = types.StringValue(fmt.Sprintf("%d", user.ID))
	model.UserID = types.Int64Value(user.UserID)
	model.Username = types.StringValue(user.Username)
	model.Home = types.StringValue(user.Home)
	model.Shell = types.StringValue(user.Shell)
	model.FullName = types.StringValue(user.FullName)
	model.Smb = types.BoolValue(user.Smb)

	switch user.UsernsIdmap.(type) {
	case string:
		model.UsernsIdmap = types.DynamicValue(types.StringValue(user.UsernsIdmap.(string)))
	case int64:
		model.UsernsIdmap = types.DynamicValue(types.Int64Value(user.UsernsIdmap.(int64)))
	}

	model.GroupID = types.Int64Value(user.Group.GroupID)

	groups, err := user.GetGroupsGIDs(s.client)
	if err != nil {
		s.handleAPIError(err)
		groups = []int64{}
	}
	model.Groups = s.listValueFrom(types.Int64Type, groups)

	model.PasswordDisabled = types.BoolValue(user.PasswordDisabled)
	model.SSHPasswordEnabled = types.BoolValue(user.SSHPasswordEnabled)
	model.Sshpubkey = types.StringPointerValue(user.Sshpubkey)
	model.Locked = types.BoolValue(user.Locked)
	model.SudoCommands = s.listValueFrom(types.StringType, user.SudoCommands)
	model.SudoCommandsNopasswd = s.listValueFrom(types.StringType, user.SudoCommandsNopasswd)
	model.Email = types.StringPointerValue(user.Email)
	model.Builtin = types.BoolValue(user.Builtin)
	model.Local = types.BoolValue(user.Local)
	model.Immutable = types.BoolValue(user.Immutable)
	model.TwofactorAuthConfigured = types.BoolValue(user.TwofactorAuthConfigured)
	model.Sid = types.StringPointerValue(user.Sid)
	// model.LastPasswordChange = types.DynamicValue(user.LastPasswordChange)
	model.PasswordAge = types.Int64PointerValue(user.PasswordAge)
	model.PasswordChangeRequired = types.BoolValue(user.PasswordChangeRequired)
	model.Roles = s.listValueFrom(types.StringType, user.Roles)
	model.APIKeys = s.listValueFrom(types.Int64Type, user.APIKeys)

	return model
}

func (s *trueNASUserService) modelToUpdateParams(model TrueNASUserResourceModel) client.UserUpdateParams {
	params := client.UserUpdateParams{
		Username:           model.Username.ValueString(),
		FullName:           model.FullName.ValueString(),
		Home:               model.Home.ValueStringPointer(),
		Shell:              model.Shell.ValueStringPointer(),
		Smb:                model.Smb.ValueBoolPointer(),
		PasswordDisabled:   model.PasswordDisabled.ValueBoolPointer(),
		SSHPasswordEnabled: model.SSHPasswordEnabled.ValueBoolPointer(),
		Sshpubkey:          model.Sshpubkey.ValueStringPointer(),
		Locked:             model.Locked.ValueBoolPointer(),
		Email:              model.Email.ValueStringPointer(),
		HomeCreate:         model.HomeCreate.ValueBoolPointer(),
		HomeMode:           model.HomeMode.ValueStringPointer(),
		Password:           model.Password.ValueStringPointer(),
		RandomPassword:     model.RandomPassword.ValueBoolPointer(),
	}

	if !model.GroupID.IsNull() {
		id, err := s.client.GroupGetID(model.GroupID.ValueInt64())
		if err != nil {
			s.handleAPIError(err)
		}
		params.Group = &id
	}

	if !model.UsernsIdmap.IsNull() {
		switch value := model.UsernsIdmap.UnderlyingValue().(type) {
		case types.Number:
			params.UsernsIdmap, _ = value.ValueBigFloat().Int64()
		case types.String:
			params.UsernsIdmap = value.ValueString()
		}
	}

	if !model.Groups.IsNull() {
		gids := s.int64ArrayFromList(model.Groups)
		ids := make([]int64, 0, len(gids))
		for _, gid := range gids {
			id, err := s.client.GroupGetID(gid)
			if err != nil {
				s.handleAPIError(err)
			}
			ids = append(ids, id)
		}

		params.Groups = &ids
	}

	if !model.SudoCommands.IsNull() {
		commands := s.stringArrayFromList(model.SudoCommands)
		params.SudoCommands = &commands
	}

	if !model.SudoCommandsNopasswd.IsNull() {
		commands := s.stringArrayFromList(model.SudoCommandsNopasswd)
		params.SudoCommandsNopasswd = &commands
	}

	if s.diags.HasError() {
		s.diags.AddError("Model Conversion Error", "Failed to convert TrueNASUserResourceModel to TrueNAS API user parameters")
	}

	return params
}

func (s *trueNASUserService) modelToCreateParams(model TrueNASUserResourceModel) client.UserCreateParams {
	return client.UserCreateParams{
		UserUpdateParams: s.modelToUpdateParams(model),
		UserID:           model.UserID.ValueInt64Pointer(),
		GroupCreate:      model.GroupCreate.ValueBoolPointer(),
	}
}

func (s *trueNASUserService) Get(id int64, out *TrueNASUserModel) diag.Diagnostics {
	response, err := s.client.UserGetInstance(id)
	if err != nil {
		s.handleAPIError(err)
		return s.diags
	}

	*out = s.modelFromAPI(response)
	return s.diags
}

func (s *trueNASUserService) Query(query TrueNASUserModel, out *TrueNASUserModel) diag.Diagnostics {
	query_user := client.UserQuery{
		UserID:   query.UserID.ValueInt64Pointer(),
		Username: query.Username.ValueStringPointer(),
	}
	if !query_user.IsValid() {
		s.diags.AddError(
			"Missing Required User Identifier",
			"At least one user identifier must be provided. Please specify either `uid` or `username`.",
		)
		return s.diags
	}

	users, err := s.client.UserQuery(query_user)
	if err != nil {
		s.handleAPIError(err)
		return s.diags
	}

	*out = s.modelFromAPI(users[0])
	return s.diags
}

func (s *trueNASUserService) Create(user TrueNASUserResourceModel, out *TrueNASUserModel) diag.Diagnostics {
	params := s.modelToCreateParams(user)
	if s.diags.HasError() {
		return s.diags
	}

	response, err := s.client.UserCreate(params)
	if err != nil {
		s.handleAPIError(err)
	}

	*out = s.modelFromAPI(response)
	return s.diags
}

func (s *trueNASUserService) Update(user TrueNASUserResourceModel, out *TrueNASUserModel) diag.Diagnostics {
	var userID int64
	s.diags.Append(user.GetID(&userID)...)
	if s.diags.HasError() {
		return s.diags
	}

	params := s.modelToUpdateParams(user)
	if s.diags.HasError() {
		return s.diags
	}

	response, err := s.client.UserUpdate(userID, params)
	if err != nil {
		s.handleAPIError(err)
	}

	*out = s.modelFromAPI(response)

	return s.diags
}

func (s *trueNASUserService) Delete(user TrueNASUserResourceModel) diag.Diagnostics {
	var id int64
	s.diags.Append(user.GetID(&id)...)
	if s.diags.HasError() {
		return s.diags
	}

	groupDelete := false
	if !user.GroupCreate.IsNull() {
		groupDelete = user.GroupCreate.ValueBool()
	}
	err := s.client.UserDelete(id, groupDelete)
	if err != nil {
		s.handleAPIError(err)
	}

	return s.diags
}
