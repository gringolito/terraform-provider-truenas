package truenas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

// UserGetByUsername looks up a user by username using user.query and returns the
// first match. Returns an error if no user with that username exists.
func UserGetByUsername(ctx context.Context, c client.Caller, username string) (*User, error) {
	raw, err := UserQuery(ctx, c, QueryFilter{Field: "username", Op: "=", Value: username})
	if err != nil {
		return nil, err
	}
	var results []User
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parsing user query result: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no user with username %q found", username)
	}
	return &results[0], nil
}

// ResolveUserUIDs converts a slice of internal API user IDs to Unix UIDs via a
// single batched user.query call. Returns an empty slice when userIDs is empty.
func ResolveUserUIDs(ctx context.Context, c client.Caller, userIDs []int64) ([]int64, error) {
	if len(userIDs) == 0 {
		return []int64{}, nil
	}
	ids := make([]any, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id
	}
	raw, err := UserQuery(ctx, c, QueryFilter{Field: "id", Op: "in", Value: ids})
	if err != nil {
		return nil, fmt.Errorf("querying users: %w", err)
	}
	var users []struct {
		Id  int64 `json:"id"`
		Uid int64 `json:"uid"`
	}
	if err := json.Unmarshal(raw, &users); err != nil {
		return nil, fmt.Errorf("parsing user query result: %w", err)
	}
	uidByID := make(map[int64]int64, len(users))
	for _, u := range users {
		uidByID[u.Id] = u.Uid
	}
	uids := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if uid, ok := uidByID[id]; ok {
			uids = append(uids, uid)
		}
	}
	return uids, nil
}

// ResolveUserIDByUID resolves a Unix UID to the internal user API ID via user.query.
// Returns an error if no user with that UID exists.
func ResolveUserIDByUID(ctx context.Context, c client.Caller, uid int64) (int64, error) {
	raw, err := UserQuery(ctx, c, QueryFilter{Field: "uid", Op: "=", Value: uid})
	if err != nil {
		return 0, fmt.Errorf("querying user by UID %d: %w", uid, err)
	}
	var users []struct {
		Id int64 `json:"id"`
	}
	if err := json.Unmarshal(raw, &users); err != nil {
		return 0, fmt.Errorf("parsing user query result: %w", err)
	}
	if len(users) == 0 {
		return 0, fmt.Errorf("no user with UID %d found", uid)
	}
	return users[0].Id, nil
}
