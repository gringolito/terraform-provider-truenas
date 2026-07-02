package truenas

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
)

// PoolDatasetPathParts splits a dataset's full ZFS path into its pool (first
// segment) and name (last segment) — derived, read-only views, never
// independent create/import inputs (see CONTEXT.md "Dataset path").
func PoolDatasetPathParts(path string) (pool, name string) {
	idx := strings.Index(path, "/")
	if idx < 0 {
		return path, path
	}
	pool = path[:idx]
	name = path[strings.LastIndex(path, "/")+1:]
	return pool, name
}

// RequireFilesystemDataset returns a clear, actionable error if ds is not a
// FILESYSTEM-type dataset. truenas_dataset manages FILESYSTEM datasets only
// (see CONTEXT.md "Dataset type"); VOLUME datasets are out of v0.1 scope.
func RequireFilesystemDataset(ds *PoolDataset) error {
	if ds.Type != "FILESYSTEM" {
		return fmt.Errorf("dataset %q is a %s dataset; truenas_dataset only manages FILESYSTEM-type datasets", ds.Id, ds.Type)
	}
	return nil
}

// PoolDatasetGetFilesystem fetches the dataset at path and verifies it is a
// FILESYSTEM-type dataset, returning a clear, actionable error otherwise.
// Shared by the resource's Read/Import and the data source's Read.
func PoolDatasetGetFilesystem(ctx context.Context, c client.Caller, path string) (*PoolDataset, error) {
	ds, err := PoolDatasetGetInstance(ctx, c, path)
	if err != nil {
		return nil, err
	}
	if err := RequireFilesystemDataset(ds); err != nil {
		return nil, err
	}
	return ds, nil
}

// PoolDatasetProperties holds the mutable ZFS dataset properties extracted
// from a PoolDataset's {value, rawvalue, parsed, source, source_info} shape
// into their Go-native, accepts-compatible representation.
type PoolDatasetProperties struct {
	Comments      *string
	Compression   *string
	Sync          *string
	Atime         *string
	Exec          *string
	Readonly      *string
	Deduplication *string
	SnapDir       *string
	RecordSize    *string

	Quota          *int64
	Refquota       *int64
	Reservation    *int64
	Refreservation *int64
	Copies         *int64

	// *String mirror TrueNAS's own human-formatted rendering (e.g. "20 GiB")
	// of the corresponding size property, straight from ZFSProperty.Value.
	// Read-only convenience fields — pool.dataset.create/.update do not
	// accept this shape as input (see docs/adr/0007).
	QuotaString          *string
	RefquotaString       *string
	ReservationString    *string
	RefreservationString *string
}

// ExtractPoolDatasetProperties extracts the mutable dataset properties from a
// PoolDataset's ZFS property shape into Go-native values.
//
// Which of ZFSProperty's `value`/`parsed` representations to read is driven
// by each property's accepts type, not guessable from the read shape alone:
// `value` mirrors the exact string/enum casing pool.dataset.create/.update
// accept, while `parsed` is a Python-native convenience type that only
// matches the accepts shape for integer properties (e.g. quota's `value` is
// human-formatted "20 GiB" while its accepts type is a raw byte integer).
// Getting this wrong silently breaks "empty plan after apply" for that
// property. See docs/adr/0007-dataset-codegen-extensions.md.
//
// Two further live-box-observed quirks (not documented in the registry
// schema) are handled here:
//   - `comments` is not returned as a top-level ZFS property at all; it is
//     stored and returned as a ZFS user property under
//     user_properties["comments"], despite pool.dataset.create/.update
//     accepting it as a plain top-level "comments" argument.
//   - For integer properties whose ZFS meaning of "0"/"none" is the same as
//     "unset" (quota, refquota, reservation, refreservation), TrueNAS reports
//     `parsed: null` (and `value: null`) rather than `parsed: 0` — only
//     `rawvalue` ("0") carries the number. Falling back to `rawvalue` there
//     keeps an explicit 0 in config from drifting to null forever.
func ExtractPoolDatasetProperties(ds *PoolDataset) (*PoolDatasetProperties, error) {
	p := &PoolDatasetProperties{
		Comments:      extractCommentsUserProperty(ds),
		Compression:   ds.Compression.Value,
		Sync:          ds.Sync.Value,
		Atime:         ds.Atime.Value,
		Exec:          ds.Exec.Value,
		Readonly:      ds.Readonly.Value,
		Deduplication: ds.Deduplication.Value,
		SnapDir:       ds.Snapdir.Value,
		RecordSize:    ds.Recordsize.Value,

		QuotaString:          ds.Quota.Value,
		RefquotaString:       ds.Refquota.Value,
		ReservationString:    ds.Reservation.Value,
		RefreservationString: ds.Refreservation.Value,
	}

	var err error
	if p.Quota, err = zfsPropertyInt(ds.Quota); err != nil {
		return nil, fmt.Errorf("quota: %w", err)
	}
	if p.Refquota, err = zfsPropertyInt(ds.Refquota); err != nil {
		return nil, fmt.Errorf("refquota: %w", err)
	}
	if p.Reservation, err = zfsPropertyInt(ds.Reservation); err != nil {
		return nil, fmt.Errorf("reservation: %w", err)
	}
	if p.Refreservation, err = zfsPropertyInt(ds.Refreservation); err != nil {
		return nil, fmt.Errorf("refreservation: %w", err)
	}
	if p.Copies, err = zfsPropertyInt(ds.Copies); err != nil {
		return nil, fmt.Errorf("copies: %w", err)
	}
	return p, nil
}

// extractCommentsUserProperty reads the "comments" ZFS user property. It
// also checks the top-level `comments` field first in case a future/other
// TrueNAS version does return it there as the registry schema declares.
func extractCommentsUserProperty(ds *PoolDataset) *string {
	if ds.Comments.Value != nil {
		return ds.Comments.Value
	}
	raw, ok := ds.UserProperties["comments"]
	if !ok {
		return nil
	}
	var prop ZFSProperty
	if err := json.Unmarshal(raw, &prop); err != nil {
		return nil
	}
	return prop.Value
}

// zfsPropertyInt decodes a ZFS integer property's effective value: `parsed`
// when present, else `rawvalue` (see the "0"/"none" quirk documented on
// ExtractPoolDatasetProperties). Returns nil (not an error) when neither
// representation carries a value.
func zfsPropertyInt(p ZFSProperty) (*int64, error) {
	if len(p.Parsed) > 0 && string(p.Parsed) != "null" {
		var n int64
		if err := json.Unmarshal(p.Parsed, &n); err != nil {
			return nil, fmt.Errorf("parsing ZFS property parsed value %s: %w", p.Parsed, err)
		}
		return &n, nil
	}
	if p.RawValue == nil {
		return nil, nil
	}
	if n, err := strconv.ParseInt(*p.RawValue, 10, 64); err == nil {
		return &n, nil
	}
	// rawvalue isn't a clean integer either — genuinely absent, not an error.
	return nil, nil
}
