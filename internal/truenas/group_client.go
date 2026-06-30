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

// ResolveGroupIDsByGIDs converts Unix GIDs to internal group API IDs via a
// single batched group.query call. Returns an empty slice when gids is empty.
func ResolveGroupIDsByGIDs(ctx context.Context, c client.Caller, gids []int64) ([]int64, error) {
	if len(gids) == 0 {
		return []int64{}, nil
	}
	vals := make([]any, len(gids))
	for i, gid := range gids {
		vals[i] = gid
	}
	raw, err := GroupQuery(ctx, c, QueryFilter{Field: "gid", Op: "in", Value: vals})
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
	apiIDByGID := make(map[int64]int64, len(groups))
	for _, g := range groups {
		apiIDByGID[g.Gid] = g.Id
	}
	apiIDs := make([]int64, 0, len(gids))
	for _, gid := range gids {
		if id, ok := apiIDByGID[gid]; ok {
			apiIDs = append(apiIDs, id)
		}
	}
	return apiIDs, nil
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
