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

func TestGroupCreate(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.create": json.RawMessage(`42`),
		},
	}
	id, err := truenas.GroupCreate(context.Background(), fake, truenas.GroupCreateArgs{Name: "testgroup"})
	if err != nil {
		t.Fatalf("GroupCreate: %v", err)
	}
	if id != 42 {
		t.Errorf("id: got %d, want 42", id)
	}
}

func TestGroupGetInstance(t *testing.T) {
	payload := `{
		"id": 42,
		"gid": 1000,
		"name": "testgroup",
		"builtin": false,
		"sudo_commands": [],
		"sudo_commands_nopasswd": [],
		"smb": true,
		"userns_idmap": null,
		"group": "testgroup",
		"local": true,
		"sid": null,
		"roles": [],
		"users": [1, 2],
		"immutable": false
	}`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.get_instance": json.RawMessage(payload),
		},
	}
	g, err := truenas.GroupGetInstance(context.Background(), fake, 42)
	if err != nil {
		t.Fatalf("GroupGetInstance: %v", err)
	}
	if g.Id != 42 {
		t.Errorf("Id: got %d, want 42", g.Id)
	}
	if g.Gid != 1000 {
		t.Errorf("Gid: got %d, want 1000", g.Gid)
	}
	if g.Name != "testgroup" {
		t.Errorf("Name: got %q, want %q", g.Name, "testgroup")
	}
	if g.Builtin {
		t.Error("Builtin: got true, want false")
	}
	if !g.Local {
		t.Error("Local: got false, want true")
	}
	if g.Sid != nil {
		t.Errorf("Sid: got %v, want nil", g.Sid)
	}
	if len(g.Users) != 2 {
		t.Errorf("Users: got %v, want [1, 2]", g.Users)
	}
}

func TestGroupUpdate(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.update": json.RawMessage(`42`),
		},
	}
	smb := true
	_, err := truenas.GroupUpdate(context.Background(), fake, 42, truenas.GroupUpdateArgs{
		Name:         "renamed",
		Smb:          &smb,
		SudoCommands: []string{},
	})
	if err != nil {
		t.Fatalf("GroupUpdate: %v", err)
	}
}

func TestGroupDelete(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.delete": json.RawMessage(`42`),
		},
	}
	if err := truenas.GroupDelete(context.Background(), fake, 42); err != nil {
		t.Fatalf("GroupDelete: %v", err)
	}
}

func TestGroupGetByName(t *testing.T) {
	payload := `[{"id":42,"gid":1000,"name":"testgroup","builtin":false,
		"sudo_commands":[],"sudo_commands_nopasswd":[],"smb":true,"userns_idmap":null,
		"group":"testgroup","local":true,"sid":null,"roles":[],"users":[],"immutable":false}]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.query": json.RawMessage(payload),
		},
	}
	g, err := truenas.GroupGetByName(context.Background(), fake, "testgroup")
	if err != nil {
		t.Fatalf("GroupGetByName: %v", err)
	}
	if g.Id != 42 {
		t.Errorf("Id: got %d, want 42", g.Id)
	}
	if g.Name != "testgroup" {
		t.Errorf("Name: got %q, want %q", g.Name, "testgroup")
	}
}

func TestGroupGetByName_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"group.query": json.RawMessage(`[]`),
		},
	}
	_, err := truenas.GroupGetByName(context.Background(), fake, "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
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

func TestGroupGetInstance_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Errors: map[string]error{
			"group.get_instance": &client.APIError{ErrName: "MatchNotFound"},
		},
	}
	_, err := truenas.GroupGetInstance(context.Background(), fake, 99)
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
