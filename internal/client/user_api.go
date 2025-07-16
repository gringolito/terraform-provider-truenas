package client

import (
	"encoding/json"
)

type UserQueryResultItem struct {
	ID                      int64            `json:"id"`
	UserID                  int64            `json:"uid"`
	Smbhash                 *string          `json:"smbhash"`
	Home                    string           `json:"home"`
	Shell                   string           `json:"shell"`
	FullName                string           `json:"full_name"`
	Smb                     bool             `json:"smb"`
	UsernsIdmap             any              `json:"userns_idmap"`
	Group                   UserPrimaryGroup `json:"group"`
	Groups                  []int64          `json:"groups"`
	PasswordDisabled        bool             `json:"password_disabled"`
	SSHPasswordEnabled      bool             `json:"ssh_password_enabled"`
	Sshpubkey               *string          `json:"sshpubkey"`
	Locked                  bool             `json:"locked"`
	SudoCommands            []string         `json:"sudo_commands"`
	SudoCommandsNopasswd    []string         `json:"sudo_commands_nopasswd"`
	Email                   *string          `json:"email"`
	Username                string           `json:"username"`
	Unixhash                *string          `json:"unixhash"`
	Builtin                 bool             `json:"builtin"`
	Local                   bool             `json:"local"`
	Immutable               bool             `json:"immutable"`
	TwofactorAuthConfigured bool             `json:"twofactor_auth_configured"`
	Sid                     *string          `json:"sid"`
	LastPasswordChange      any              `json:"last_password_change"`
	PasswordAge             *int64           `json:"password_age"`
	PasswordHistory         any              `json:"password_history"`
	PasswordChangeRequired  bool             `json:"password_change_required"`
	Roles                   []string         `json:"roles"`
	APIKeys                 []int64          `json:"api_keys"`
}

type UserPrimaryGroup struct {
	ID                   int64    `json:"id"`
	GroupID              int64    `json:"bsdgrp_gid"`
	Group                string   `json:"bsdgrp_group"`
	Builtin              bool     `json:"bsdgrp_builtin"`
	Smb                  bool     `json:"bsdgrp_smb"`
	SudoCommands         []string `json:"bsdgrp_sudo_commands"`
	SudoCommandsNopasswd []string `json:"bsdgrp_sudo_commands_nopasswd"`
	UsernsIdmap          any      `json:"bsdgrp_userns_idmap"`
}

func (g *UserQueryResultItem) GetGroupsGIDs(c *Client) ([]int64, error) {
	var groupsGIDs []int64
	for _, groupid := range g.Groups {
		gid, err := c.GroupGetGID(groupid)
		if err != nil {
			return nil, err
		}
		groupsGIDs = append(groupsGIDs, gid)
	}

	return groupsGIDs, nil
}

type UserQuery struct {
	UserID   *int64  `query:"uid"`
	Username *string `query:"username"`
}

func (q *UserQuery) IsValid() bool {
	return q.UserID != nil || q.Username != nil
}

func (q *UserQuery) ToParams() []any {
	params := []any{}
	if q.UserID != nil {
		params = append(params, []any{"uid", "=", *q.UserID})
	}
	if q.Username != nil {
		params = append(params, []any{"username", "=", *q.Username})
	}
	return params
}

type UserCreateParams struct {
	UserUpdateParams
	UserID      *int64 `json:"uid,omitempty"`
	GroupCreate *bool  `json:"group_create,omitempty"`
}

type UserUpdateParams struct {
	Username             string    `json:"username"`
	FullName             string    `json:"full_name"`
	Home                 *string   `json:"home,omitempty"`
	Shell                *string   `json:"shell,omitempty"`
	Smb                  *bool     `json:"smb,omitempty"`
	UsernsIdmap          any       `json:"usernd_idmap,omitempty"`
	Group                *int64    `json:"group,omitempty"`
	Groups               *[]int64  `json:"groups,omitempty"`
	PasswordDisabled     *bool     `json:"password_disabled,omitempty"`
	SSHPasswordEnabled   *bool     `json:"ssh_password_enabled,omitempty"`
	Sshpubkey            *string   `json:"sshpubkey,omitempty"`
	Locked               *bool     `json:"locked,omitempty"`
	SudoCommands         *[]string `json:"sudo_commands,omitempty"`
	SudoCommandsNopasswd *[]string `json:"sudo_commands_nopasswd,omitempty"`
	Email                *string   `json:"email,omitempty"`
	HomeCreate           *bool     `json:"home_create,omitempty"`
	HomeMode             *string   `json:"home_mode,omitempty"`
	Password             *string   `json:"password,omitempty"`
	RandomPassword       *bool     `json:"random_password,omitempty"`
}

func (c *Client) UserGetUID(id int64) (int64, error) {
	user, err := c.UserGetInstance(id)
	return user.UserID, err
}

func (c *Client) UserGetID(uid int64) (int64, error) {
	users, err := c.UserQuery(UserQuery{UserID: &uid})
	if err != nil {
		return 0, err
	}

	user := users[0]
	return user.ID, err
}

func (c *Client) UserQuery(query UserQuery) ([]UserQueryResultItem, error) {
	response, err := c.Call("user.query", []any{query.ToParams()})
	if err != nil {
		return nil, APIError{"Failed to Read TrueNAS user", err}
	}

	var users []UserQueryResultItem
	err = json.Unmarshal(response, &users)
	if err != nil {
		return nil, APIError{"Invalid TrueNAS user query response", err}
	}

	return users, nil
}

func (c *Client) UserGetInstance(id int64) (UserQueryResultItem, error) {
	var user UserQueryResultItem

	response, err := c.Call("user.get_instance", []any{id})
	if err != nil {
		return user, APIError{"Failed to Read TrueNAS user", err}
	}

	err = json.Unmarshal(response, &user)
	if err != nil {
		return user, APIError{"Invalid TrueNAS user get-instance response", err}
	}

	return user, nil
}

func (c *Client) UserCreate(user UserCreateParams) (UserQueryResultItem, error) {
	var response UserQueryResultItem

	params, err := c.toAPIParams(user)
	if err != nil {
		return response, APIError{"Failed to parse TrueNAS user params", err}
	}

	createData, err := c.Call("user.create", []any{params})
	if err != nil {
		return response, APIError{"Failed to Create TrueNAS user", err}
	}

	err = json.Unmarshal(createData, &response)
	if err != nil {
		return response, APIError{"Invalid TrueNAS user create response", err}
	}

	return response, nil
}

func (c *Client) UserUpdate(id int64, user UserUpdateParams) (UserQueryResultItem, error) {
	var response UserQueryResultItem

	params, err := c.toAPIParams(user)
	if err != nil {
		return response, APIError{"Failed to parse TrueNAS user params", err}
	}

	updateData, err := c.Call("user.update", []any{id, params})
	if err != nil {
		return response, APIError{"Failed to Update TrueNAS user", err}
	}

	err = json.Unmarshal(updateData, &response)
	if err != nil {
		return response, APIError{"Invalid TrueNAS user update response", err}
	}

	return response, nil
}

func (c *Client) UserDelete(id int64, deleteGroup bool) error {
	deleteOptions := map[string]any{"delete_group": deleteGroup}
	_, err := c.Call("user.delete", []any{id, deleteOptions})
	if err != nil {
		return APIError{"Failed to Delete TrueNAS user", err}
	}

	return nil
}
