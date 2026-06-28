package truenas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

// GroupGetByName looks up a group by name using group.query and returns the
// first match. Returns an error if no group with that name exists.
func GroupGetByName(ctx context.Context, c client.Caller, name string) (*Group, error) {
	raw, err := GroupQuery(ctx, c, QueryFilter{Field: "name", Op: "=", Value: name})
	if err != nil {
		return nil, err
	}
	var results []Group
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parsing group query result: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no group with name %q found", name)
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
