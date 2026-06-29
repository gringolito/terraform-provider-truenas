package truenas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

// userRaw is a safe decode target for API responses that avoids the
// last_password_change field, which the TrueNAS API returns as a MongoDB
// {"$date": ...} object that cannot be unmarshaled into *string.
type userRaw struct {
	Id                      int64                      `json:"id"`
	Uid                     int64                      `json:"uid"`
	Username                json.RawMessage            `json:"username"`
	Unixhash                *string                    `json:"unixhash"`
	Smbhash                 *string                    `json:"smbhash"`
	Home                    string                     `json:"home,omitempty"`
	Shell                   string                     `json:"shell,omitempty"`
	FullName                string                     `json:"full_name"`
	Builtin                 bool                       `json:"builtin"`
	Smb                     bool                       `json:"smb,omitempty"`
	UsernsIdmap             json.RawMessage            `json:"userns_idmap,omitempty"`
	Group                   map[string]json.RawMessage `json:"group"`
	Groups                  []int64                    `json:"groups,omitempty"`
	PasswordDisabled        bool                       `json:"password_disabled,omitempty"`
	SshPasswordEnabled      bool                       `json:"ssh_password_enabled,omitempty"`
	Sshpubkey               *string                    `json:"sshpubkey,omitempty"`
	Locked                  bool                       `json:"locked,omitempty"`
	SudoCommands            []string                   `json:"sudo_commands,omitempty"`
	SudoCommandsNopasswd    []string                   `json:"sudo_commands_nopasswd,omitempty"`
	Email                   *string                    `json:"email,omitempty"`
	Local                   bool                       `json:"local"`
	Immutable               bool                       `json:"immutable"`
	TwofactorAuthConfigured bool                       `json:"twofactor_auth_configured"`
	Sid                     *string                    `json:"sid"`
	LastPasswordChange      json.RawMessage            `json:"last_password_change"` // {"$date":...} object — kept as raw
	Roles                   []string                   `json:"roles"`
	ApiKeys                 []int64                    `json:"api_keys"`
}

func userRawToUser(r *userRaw) *User {
	return &User{
		Id:                      r.Id,
		Uid:                     r.Uid,
		Username:                r.Username,
		Unixhash:                r.Unixhash,
		Smbhash:                 r.Smbhash,
		Home:                    r.Home,
		Shell:                   r.Shell,
		FullName:                r.FullName,
		Builtin:                 r.Builtin,
		Smb:                     r.Smb,
		UsernsIdmap:             r.UsernsIdmap,
		Group:                   r.Group,
		Groups:                  r.Groups,
		PasswordDisabled:        r.PasswordDisabled,
		SshPasswordEnabled:      r.SshPasswordEnabled,
		Sshpubkey:               r.Sshpubkey,
		Locked:                  r.Locked,
		SudoCommands:            r.SudoCommands,
		SudoCommandsNopasswd:    r.SudoCommandsNopasswd,
		Email:                   r.Email,
		Local:                   r.Local,
		Immutable:               r.Immutable,
		TwofactorAuthConfigured: r.TwofactorAuthConfigured,
		Sid:                     r.Sid,
		Roles:                   r.Roles,
		ApiKeys:                 r.ApiKeys,
	}
}

// UserGetInstanceSafe fetches a user by internal API ID and safely decodes the
// response, handling last_password_change as raw JSON to avoid type errors from
// the MongoDB {"$date": ...} object the TrueNAS API returns.
func UserGetInstanceSafe(ctx context.Context, c client.Caller, id int64) (*User, error) {
	raw, err := c.Call(ctx, "user.get_instance", []any{id})
	if err != nil {
		return nil, err
	}
	var r userRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decoding user.get_instance response: %w", err)
	}
	return userRawToUser(&r), nil
}

// UserGetByUsername looks up a user by username using user.query and returns the
// first match. Returns an error if no user with that username exists.
func UserGetByUsername(ctx context.Context, c client.Caller, username string) (*User, error) {
	raw, err := UserQuery(ctx, c, QueryFilter{Field: "username", Op: "=", Value: username})
	if err != nil {
		return nil, err
	}
	var results []userRaw
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parsing user query result: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no user with username %q found", username)
	}
	return userRawToUser(&results[0]), nil
}

// UserCreateSafe calls user.create and safely decodes the response.
func UserCreateSafe(ctx context.Context, c client.Caller, args UserCreateArgs) (*User, error) {
	raw, err := c.Call(ctx, "user.create", []any{args})
	if err != nil {
		return nil, err
	}
	var r userRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decoding user.create response: %w", err)
	}
	return userRawToUser(&r), nil
}

// UserUpdateSafe calls user.update and safely decodes the response.
func UserUpdateSafe(ctx context.Context, c client.Caller, id int64, args UserUpdateArgs) (*User, error) {
	raw, err := c.Call(ctx, "user.update", []any{id, args})
	if err != nil {
		return nil, err
	}
	var r userRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("decoding user.update response: %w", err)
	}
	return userRawToUser(&r), nil
}

// ResolveGroupGIDs converts a slice of internal group API IDs to Unix GIDs via a
// single batched group.query call. Returns an empty slice when groupIDs is empty.
func ResolveGroupGIDs(ctx context.Context, c client.Caller, groupIDs []int64) ([]int64, error) {
	if len(groupIDs) == 0 {
		return []int64{}, nil
	}
	ids := make([]any, len(groupIDs))
	for i, id := range groupIDs {
		ids[i] = id
	}
	raw, err := GroupQuery(ctx, c, QueryFilter{Field: "id", Op: "in", Value: ids})
	if err != nil {
		return nil, fmt.Errorf("querying groups: %w", err)
	}
	var groups []struct {
		Id  int64 `json:"id"`
		Gid int64 `json:"gid"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		return nil, fmt.Errorf("parsing group query result: %w", err)
	}
	gidByID := make(map[int64]int64, len(groups))
	for _, g := range groups {
		gidByID[g.Id] = g.Gid
	}
	gids := make([]int64, 0, len(groupIDs))
	for _, id := range groupIDs {
		if gid, ok := gidByID[id]; ok {
			gids = append(gids, gid)
		}
	}
	return gids, nil
}

// ResolveGroupIDByGID resolves a Unix GID to the internal group API ID via group.query.
// Returns an error if no group with that GID exists.
func ResolveGroupIDByGID(ctx context.Context, c client.Caller, gid int64) (int64, error) {
	raw, err := GroupQuery(ctx, c, QueryFilter{Field: "gid", Op: "=", Value: gid})
	if err != nil {
		return 0, fmt.Errorf("querying group by GID %d: %w", gid, err)
	}
	var groups []struct {
		Id  int64 `json:"id"`
		Gid int64 `json:"gid"`
	}
	if err := json.Unmarshal(raw, &groups); err != nil {
		return 0, fmt.Errorf("parsing group query result: %w", err)
	}
	if len(groups) == 0 {
		return 0, fmt.Errorf("no group with GID %d found", gid)
	}
	return groups[0].Id, nil
}
