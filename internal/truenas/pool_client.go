package truenas

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

type PoolAutotrim struct {
	Parsed string `json:"parsed"`
}

type PoolScan struct {
	Function       string    `json:"function"`
	State          string    `json:"state"`
	StartTime      DateTime  `json:"start_time"`
	EndTime        DateTime  `json:"end_time"`
	Percentage     float64   `json:"percentage"`
	BytesToProcess int64     `json:"bytes_to_process"`
	BytesProcessed int64     `json:"bytes_processed"`
	BytesIssued    int64     `json:"bytes_issued"`
	Errors         int64     `json:"errors"`
	Pause          *DateTime `json:"pause"`
	TotalSecsLeft  *int64    `json:"total_secs_left"`
}

type PoolDetail struct {
	Id              int64        `json:"id"`
	Name            string       `json:"name"`
	Guid            string       `json:"guid"`
	Status          string       `json:"status"`
	Path            string       `json:"path"`
	Healthy         bool         `json:"healthy"`
	Warning         bool         `json:"warning"`
	IsUpgraded      bool         `json:"is_upgraded,omitempty"`
	StatusCode      *string      `json:"status_code"`
	StatusDetail    *string      `json:"status_detail"`
	Size            *int64       `json:"size"`
	Allocated       *int64       `json:"allocated"`
	Free            *int64       `json:"free"`
	Freeing         *int64       `json:"freeing"`
	DedupTableSize  *int64       `json:"dedup_table_size"`
	DedupTableQuota *string      `json:"dedup_table_quota"`
	Fragmentation   *string      `json:"fragmentation"`
	Autotrim        PoolAutotrim `json:"autotrim"`
	Scan            *PoolScan    `json:"scan"`
}

// PoolGetByName looks up a pool by name using pool.query and returns the
// first match. Returns an error if no pool with that name exists.
func PoolGetByName(ctx context.Context, c client.Caller, name string) (*PoolDetail, error) {
	raw, err := PoolQuery(ctx, c, QueryFilter{Field: "name", Op: "=", Value: name})
	if err != nil {
		return nil, err
	}
	var results []PoolDetail
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("parsing pool query result: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no pool with name %q found", name)
	}
	return &results[0], nil
}
