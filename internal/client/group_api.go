package client

import (
	"encoding/json"
)

type GroupQueryResultItem struct {
	ID                   int64    `json:"id"`
	GroupID              int64    `json:"gid"`
	Name                 string   `json:"name"`
	SudoCommands         []string `json:"sudo_commands"`
	SudoCommandsNopasswd []string `json:"sudo_commands_nopasswd"`
	Smb                  bool     `json:"smb"`
	UsernsIdmap          any      `json:"userns_idmap"`
	Users                []int64  `json:"users"`
	Group                string   `json:"group"`
	Builtin              bool     `json:"builtin"`
	Local                bool     `json:"local"`
	Sid                  *string  `json:"sid"`
	Roles                []string `json:"roles"`
}

func (g *GroupQueryResultItem) GetUsersUIDs(c *Client) ([]int64, error) {
	var usersUIDs []int64
	for _, userid := range g.Users {
		uid, err := c.UserGetUID(userid)
		if err != nil {
			return nil, err
		}
		usersUIDs = append(usersUIDs, uid)
	}

	return usersUIDs, nil
}

type GroupQuery struct {
	GroupID *int64  `query:"gid"`
	Name    *string `query:"name"`
}

func (q *GroupQuery) IsValid() bool {
	return q.GroupID != nil || q.Name != nil
}

func (q *GroupQuery) ToParams() []any {
	params := []any{}
	if q.GroupID != nil {
		params = append(params, []any{"gid", "=", *q.GroupID})
	}
	if q.Name != nil {
		params = append(params, []any{"name", "=", *q.Name})
	}
	return params
}

type GroupCreateParams struct {
	GroupUpdateParams
	GroupID *int64 `json:"gid"`
}

type GroupUpdateParams struct {
	Name                 string    `json:"name"`
	SudoCommands         *[]string `json:"sudo_commands"`
	SudoCommandsNopasswd *[]string `json:"sudo_commands_nopasswd"`
	Smb                  *bool     `json:"smb"`
	UsernsIdmap          any       `json:"userns_idmap"`
	Users                *[]int64  `json:"users"`
}

func (c *Client) GroupGetGID(id int64) (int64, error) {
	group, err := c.GroupGetInstance(id)
	return group.GroupID, err
}

func (c *Client) GroupGetID(gid int64) (int64, error) {
	groups, err := c.GroupQuery(GroupQuery{GroupID: &gid})
	if err != nil {
		return 0, err
	}

	group := groups[0]
	return group.ID, err
}

func (c *Client) GroupQuery(query GroupQuery) ([]GroupQueryResultItem, error) {
	response, err := c.Call("group.query", []any{query.ToParams()})
	if err != nil {
		return nil, APIError{"Failed to Read TrueNAS group", err}
	}

	var groups []GroupQueryResultItem
	err = json.Unmarshal(response, &groups)
	if err != nil {
		return nil, APIError{"Invalid TrueNAS group query response", err}
	}

	return groups, nil
}

func (c *Client) GroupGetInstance(id int64) (GroupQueryResultItem, error) {
	var group GroupQueryResultItem

	response, err := c.Call("group.get_instance", []any{id})
	if err != nil {
		return group, APIError{"Failed to Read TrueNAS group", err}
	}

	err = json.Unmarshal(response, &group)
	if err != nil {
		return group, APIError{"Invalid TrueNAS group get-instance response", err}
	}

	return group, nil
}

func (c *Client) GroupCreate(group GroupCreateParams) (int64, error) {
	params, err := c.toAPIParams(group)
	if err != nil {
		return 0, APIError{"Failed to parse TrueNAS group params", err}
	}

	response, err := c.Call("group.create", []any{params})
	if err != nil {
		return 0, APIError{"Failed to Create TrueNAS group", err}
	}

	var id int64
	err = json.Unmarshal(response, &id)
	if err != nil {
		return 0, APIError{"Invalid TrueNAS group create response", err}
	}

	return id, nil
}

func (c *Client) GroupUpdate(id int64, group GroupUpdateParams) error {
	params, err := c.toAPIParams(group)
	if err != nil {
		return APIError{"Failed to parse TrueNAS group params", err}
	}

	_, err = c.Call("group.update", []any{id, params})
	if err != nil {
		return APIError{"Failed to Update TrueNAS group", err}
	}

	return nil
}

func (c *Client) GroupDelete(id int64) error {
	_, err := c.Call("group.delete", []any{id})
	if err != nil {
		return APIError{"Failed to Delete TrueNAS group", err}
	}

	return nil
}
