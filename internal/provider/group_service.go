package provider

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TrueNASGroupModel describes the schema and state for a TrueNAS group.
type TrueNASGroupModel struct {
	ID                   types.String  `tfsdk:"id"`
	GroupID              types.Int64   `tfsdk:"gid"`
	Name                 types.String  `tfsdk:"name"`
	SudoCommands         types.List    `tfsdk:"sudo_commands"`
	SudoCommandsNopasswd types.List    `tfsdk:"sudo_commands_nopasswd"`
	Smb                  types.Bool    `tfsdk:"smb"`
	UsernsIdmap          types.Dynamic `tfsdk:"userns_idmap"`
	Users                types.List    `tfsdk:"users"`
	Group                types.String  `tfsdk:"group"`
	Builtin              types.Bool    `tfsdk:"builtin"`
	Local                types.Bool    `tfsdk:"local"`
	Sid                  types.String  `tfsdk:"sid"`
	Roles                types.List    `tfsdk:"roles"`
}

func (g *TrueNASGroupModel) GetID(id *int64) diag.Diagnostics {
	var diags diag.Diagnostics
	parsedID, err := strconv.ParseInt(g.ID.ValueString(), 10, 0)
	if err != nil {
		diags.AddError("Failed to get TrueNAS Group ID", err.Error())
	}

	*id = parsedID
	return diags
}

type trueNASGroupService struct {
	trueNASService
}

func NewTrueNASGroupService(ctx context.Context, client *client.Client) trueNASGroupService {
	return trueNASGroupService{trueNASService{ctx, client, diag.Diagnostics{}}}
}

func (s *trueNASGroupService) modelFromAPI(group client.GroupQueryResultItem) TrueNASGroupModel {
	var model TrueNASGroupModel

	model.ID = types.StringValue(fmt.Sprintf("%d", group.ID))
	model.GroupID = types.Int64Value(group.GroupID)
	model.Name = types.StringValue(group.Name)
	model.SudoCommands = s.listValueFrom(types.StringType, group.SudoCommands)
	model.SudoCommandsNopasswd = s.listValueFrom(types.StringType, group.SudoCommandsNopasswd)
	model.Smb = types.BoolValue(group.Smb)

	switch group.UsernsIdmap.(type) {
	case string:
		model.UsernsIdmap = types.DynamicValue(types.StringValue(group.UsernsIdmap.(string)))
	case int64:
		model.UsernsIdmap = types.DynamicValue(types.Int64Value(group.UsernsIdmap.(int64)))
	}

	users, err := group.GetUsersUIDs(s.client)
	if err != nil {
		s.handleAPIError(err)
		users = []int64{}
	}
	model.Users = s.listValueFrom(types.Int64Type, users)

	model.Group = types.StringValue(group.Group)
	model.Builtin = types.BoolValue(group.Builtin)
	model.Local = types.BoolValue(group.Local)
	model.Sid = types.StringPointerValue(group.Sid)
	model.Roles = s.listValueFrom(types.StringType, group.Roles)

	return model
}

func (s *trueNASGroupService) modelToUpdateParams(model TrueNASGroupModel) client.GroupUpdateParams {
	params := client.GroupUpdateParams{
		Name: model.Name.ValueString(),
		Smb:  model.Smb.ValueBoolPointer(),
	}

	if !model.SudoCommands.IsNull() {
		commands := s.stringArrayFromList(model.SudoCommands)
		params.SudoCommands = &commands
	}

	if !model.SudoCommandsNopasswd.IsNull() {
		commands := s.stringArrayFromList(model.SudoCommandsNopasswd)
		params.SudoCommandsNopasswd = &commands
	}

	if !model.UsernsIdmap.IsNull() {
		switch value := model.UsernsIdmap.UnderlyingValue().(type) {
		case types.Number:
			params.UsernsIdmap, _ = value.ValueBigFloat().Int64()
		case types.String:
			params.UsernsIdmap = value.ValueString()
		}
	}

	if !model.Users.IsNull() {
		uids := s.int64ArrayFromList(model.Users)
		ids := make([]int64, 0, len(uids))
		for _, uid := range uids {
			id, err := s.client.UserGetID(uid)
			if err != nil {
				s.handleAPIError(err)
			}
			ids = append(ids, id)
		}

		params.Users = &ids
	}

	if s.diags.HasError() {
		s.diags.AddError("Model Conversion Error", "Failed to convert TrueNASGroupModel to TrueNAS API group parameters")
	}

	return params
}

func (s *trueNASGroupService) modelToCreateParams(model TrueNASGroupModel) client.GroupCreateParams {
	return client.GroupCreateParams{
		GroupUpdateParams: s.modelToUpdateParams(model),
		GroupID:           model.GroupID.ValueInt64Pointer(),
	}
}

func (s *trueNASGroupService) Get(id int64, group *TrueNASGroupModel) diag.Diagnostics {
	response, err := s.client.GroupGetInstance(id)
	if err != nil {
		s.handleAPIError(err)
		return s.diags
	}

	*group = s.modelFromAPI(response)
	return s.diags
}

func (s *trueNASGroupService) Query(group TrueNASGroupModel, out *TrueNASGroupModel) diag.Diagnostics {
	query := client.GroupQuery{
		GroupID: group.GroupID.ValueInt64Pointer(),
		Name:    group.Name.ValueStringPointer(),
	}
	if !query.IsValid() {
		s.diags.AddError(
			"Missing Required Group Identifier",
			"At least one group identifier must be provided. Please specify either `gid` or `name`.",
		)
		return s.diags
	}

	groups, err := s.client.GroupQuery(query)
	if err != nil {
		s.handleAPIError(err)
		return s.diags
	}

	*out = s.modelFromAPI(groups[0])
	return s.diags
}

func (s *trueNASGroupService) Create(group TrueNASGroupModel, id *int64) diag.Diagnostics {
	params := s.modelToCreateParams(group)
	if s.diags.HasError() {
		return s.diags
	}

	response_id, err := s.client.GroupCreate(params)
	if err != nil {
		s.handleAPIError(err)
	}

	*id = response_id
	return s.diags
}

func (s *trueNASGroupService) Update(group TrueNASGroupModel, id *int64) diag.Diagnostics {
	var groupID int64
	s.diags.Append(group.GetID(&groupID)...)
	if s.diags.HasError() {
		return s.diags
	}

	params := s.modelToUpdateParams(group)
	if s.diags.HasError() {
		return s.diags
	}

	err := s.client.GroupUpdate(groupID, params)
	if err != nil {
		s.handleAPIError(err)
	}

	*id = groupID

	return s.diags
}

func (s *trueNASGroupService) Delete(group TrueNASGroupModel) diag.Diagnostics {
	var id int64
	s.diags.Append(group.GetID(&id)...)
	if s.diags.HasError() {
		return s.diags
	}

	err := s.client.GroupDelete(id)
	if err != nil {
		s.handleAPIError(err)
	}

	return s.diags
}
