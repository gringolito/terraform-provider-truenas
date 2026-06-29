package truenas_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/client/clienttest"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

const userPayload = `{
	"id": 33,
	"uid": 950,
	"username": "truenas_admin",
	"full_name": "TrueNAS Admin",
	"builtin": false,
	"local": true,
	"immutable": false,
	"smb": true,
	"home": "/home/truenas_admin",
	"shell": "/bin/bash",
	"password_disabled": false,
	"ssh_password_enabled": false,
	"locked": false,
	"sudo_commands": [],
	"sudo_commands_nopasswd": [],
	"userns_idmap": null,
	"group": {"id": 43, "bsdgrp_gid": 950, "bsdgrp_group": "truenas_admin"},
	"groups": [40, 120],
	"email": null,
	"sshpubkey": null,
	"sid": null,
	"roles": [],
	"api_keys": [],
	"twofactor_auth_configured": false,
	"last_password_change": {"$date": 1752182156000}
}`

// TestUserGetInstance exercises the generated UserGetInstance against a payload
// whose last_password_change is the Mongo-style {"$date": ...} object TrueNAS
// returns, verifying the generated DateTime field decodes it without error.
func TestUserGetInstance(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.get_instance": json.RawMessage(userPayload),
		},
	}
	u, err := truenas.UserGetInstance(context.Background(), fake, 33)
	if err != nil {
		t.Fatalf("UserGetInstance: %v", err)
	}
	if u.Id != 33 {
		t.Errorf("Id: got %d, want 33", u.Id)
	}
	if u.Uid != 950 {
		t.Errorf("Uid: got %d, want 950", u.Uid)
	}
	var username string
	if err := json.Unmarshal(u.Username, &username); err != nil {
		t.Fatalf("Username unmarshal: %v", err)
	}
	if username != "truenas_admin" {
		t.Errorf("Username: got %q, want %q", username, "truenas_admin")
	}
	if u.FullName != "TrueNAS Admin" {
		t.Errorf("FullName: got %q, want %q", u.FullName, "TrueNAS Admin")
	}
	if len(u.Groups) != 2 {
		t.Errorf("Groups: got %v, want [40, 120]", u.Groups)
	}
	if u.LastPasswordChange == nil {
		t.Fatal("LastPasswordChange: got nil, want decoded {\"$date\":...} object")
	}
	if got := u.LastPasswordChange.UnixMilli(); got != 1752182156000 {
		t.Errorf("LastPasswordChange: got %d ms, want 1752182156000", got)
	}
}

func TestUserGetInstance_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Errors: map[string]error{
			"user.get_instance": &client.APIError{ErrName: "MatchNotFound"},
		},
	}
	_, err := truenas.UserGetInstance(context.Background(), fake, 99)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T: %v", err, err)
	}
	if apiErr.ErrName != "MatchNotFound" {
		t.Errorf("ErrName: got %q, want %q", apiErr.ErrName, "MatchNotFound")
	}
}

func TestUserGetByUsername(t *testing.T) {
	payload := `[` + userPayload + `]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.query": json.RawMessage(payload),
		},
	}
	u, err := truenas.UserGetByUsername(context.Background(), fake, "truenas_admin")
	if err != nil {
		t.Fatalf("UserGetByUsername: %v", err)
	}
	if u.Id != 33 {
		t.Errorf("Id: got %d, want 33", u.Id)
	}
	if u.Uid != 950 {
		t.Errorf("Uid: got %d, want 950", u.Uid)
	}
}

func TestUserGetByUsername_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.query": json.RawMessage(`[]`),
		},
	}
	_, err := truenas.UserGetByUsername(context.Background(), fake, "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveUserIDByUID(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.query": json.RawMessage(`[{"id":33,"uid":950}]`),
		},
	}
	id, err := truenas.ResolveUserIDByUID(context.Background(), fake, 950)
	if err != nil {
		t.Fatalf("ResolveUserIDByUID: %v", err)
	}
	if id != 33 {
		t.Errorf("id: got %d, want 33", id)
	}
}

func TestResolveUserUIDs(t *testing.T) {
	payload := `[{"id":10,"uid":1001},{"id":20,"uid":1002}]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.query": json.RawMessage(payload),
		},
	}
	uids, err := truenas.ResolveUserUIDs(context.Background(), fake, []int64{10, 20})
	if err != nil {
		t.Fatalf("ResolveUserUIDs: %v", err)
	}
	if len(uids) != 2 {
		t.Fatalf("len: got %d, want 2", len(uids))
	}
	if uids[0] != 1001 || uids[1] != 1002 {
		t.Errorf("uids: got %v, want [1001 1002]", uids)
	}
}

func TestResolveUserUIDs_Empty(t *testing.T) {
	fake := &clienttest.FakeCaller{}
	uids, err := truenas.ResolveUserUIDs(context.Background(), fake, nil)
	if err != nil {
		t.Fatalf("ResolveUserUIDs: %v", err)
	}
	if len(uids) != 0 {
		t.Errorf("expected empty, got %v", uids)
	}
}

func TestResolveUserIDByUID_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"user.query": json.RawMessage(`[]`),
		},
	}
	_, err := truenas.ResolveUserIDByUID(context.Background(), fake, 9999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
