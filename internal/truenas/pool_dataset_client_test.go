package truenas_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/client/clienttest"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

func TestPoolDatasetPathParts(t *testing.T) {
	tests := []struct {
		path     string
		wantPool string
		wantName string
	}{
		{"tank/myuser", "tank", "myuser"},
		{"tank/projects/child", "tank", "child"},
	}
	for _, tt := range tests {
		pool, name := truenas.PoolDatasetPathParts(tt.path)
		if pool != tt.wantPool || name != tt.wantName {
			t.Errorf("PoolDatasetPathParts(%q) = (%q, %q), want (%q, %q)", tt.path, pool, name, tt.wantPool, tt.wantName)
		}
	}
}

func TestRequireFilesystemDataset(t *testing.T) {
	if err := truenas.RequireFilesystemDataset(&truenas.PoolDataset{Id: "tank/fs", Type: "FILESYSTEM"}); err != nil {
		t.Errorf("expected no error for FILESYSTEM dataset, got %v", err)
	}
	err := truenas.RequireFilesystemDataset(&truenas.PoolDataset{Id: "tank/vol", Type: "VOLUME"})
	if err == nil {
		t.Fatal("expected error for VOLUME dataset, got nil")
	}
}

func TestPoolDatasetGetFilesystem(t *testing.T) {
	payload := `{"id":"tank/fs","type":"FILESYSTEM","name":"fs","pool":"tank"}`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.dataset.get_instance": json.RawMessage(payload),
		},
	}
	ds, err := truenas.PoolDatasetGetFilesystem(context.Background(), fake, "tank/fs")
	if err != nil {
		t.Fatalf("PoolDatasetGetFilesystem: %v", err)
	}
	if ds.Id != "tank/fs" {
		t.Errorf("Id: got %q, want %q", ds.Id, "tank/fs")
	}
}

func TestPoolDatasetGetFilesystem_VolumeTypeError(t *testing.T) {
	payload := `{"id":"tank/vol","type":"VOLUME","name":"vol","pool":"tank"}`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.dataset.get_instance": json.RawMessage(payload),
		},
	}
	_, err := truenas.PoolDatasetGetFilesystem(context.Background(), fake, "tank/vol")
	if err == nil {
		t.Fatal("expected error for VOLUME dataset, got nil")
	}
}

func TestPoolDatasetGetFilesystem_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Errors: map[string]error{
			"pool.dataset.get_instance": &client.APIError{ErrName: "MatchNotFound"},
		},
	}
	_, err := truenas.PoolDatasetGetFilesystem(context.Background(), fake, "tank/missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// zfsProp builds a ZFSProperty literal the way pool.dataset.get_instance
// returns it: value carries the exact create/update-compatible casing,
// parsed carries the Python-native convenience type. See docs/adr/0007.
func zfsProp(value, rawvalue string, parsed json.RawMessage) truenas.ZFSProperty {
	return truenas.ZFSProperty{
		Value:    &value,
		RawValue: &rawvalue,
		Parsed:   parsed,
	}
}

func TestExtractPoolDatasetProperties(t *testing.T) {
	ds := &truenas.PoolDataset{
		Id:            "tank/fs",
		Type:          "FILESYSTEM",
		Compression:   zfsProp("LZ4", "lz4", json.RawMessage(`"lz4"`)),
		Sync:          zfsProp("STANDARD", "standard", json.RawMessage(`"standard"`)),
		Atime:         zfsProp("ON", "on", json.RawMessage(`"on"`)),
		Exec:          zfsProp("ON", "on", json.RawMessage(`"on"`)),
		Readonly:      zfsProp("OFF", "off", json.RawMessage(`"off"`)),
		Deduplication: zfsProp("OFF", "off", json.RawMessage(`"off"`)),
		Snapdir:       zfsProp("HIDDEN", "hidden", json.RawMessage(`"hidden"`)),
		Recordsize:    zfsProp("128K", "131072", json.RawMessage(`"128K"`)),
		// On live TrueNAS, "comments" is not a top-level property; it is a ZFS
		// user property. See docs/adr/0007 and extractCommentsUserProperty.
		UserProperties: map[string]json.RawMessage{
			"comments": json.RawMessage(`{"value":"a comment","rawvalue":"a comment","parsed":"a comment","source":"LOCAL"}`),
		},
		// quota's value is human-formatted ("20 GiB") and does not round-trip
		// into the raw byte-count accepts shape; parsed does. See docs/adr/0007.
		Quota:    zfsProp("20 GiB", "21474836480", json.RawMessage(`21474836480`)),
		Refquota: zfsProp("10 GiB", "10737418240", json.RawMessage(`10737418240`)),
		// A "0"/none reservation reports parsed:null and value:null on live
		// TrueNAS; only rawvalue carries the number. See docs/adr/0007.
		Reservation:    truenas.ZFSProperty{RawValue: strPtr("0"), Parsed: json.RawMessage(`null`)},
		Refreservation: truenas.ZFSProperty{RawValue: strPtr("0"), Parsed: json.RawMessage(`null`)},
		Copies:         zfsProp("1", "1", json.RawMessage(`1`)),
	}

	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		t.Fatalf("ExtractPoolDatasetProperties: %v", err)
	}

	// String properties must read `value` (exact accepts casing), not `parsed`.
	stringChecks := map[string]struct {
		got  *string
		want string
	}{
		"comments":      {props.Comments, "a comment"},
		"compression":   {props.Compression, "LZ4"},
		"sync":          {props.Sync, "STANDARD"},
		"atime":         {props.Atime, "ON"},
		"exec":          {props.Exec, "ON"},
		"readonly":      {props.Readonly, "OFF"},
		"deduplication": {props.Deduplication, "OFF"},
		"snap_dir":      {props.SnapDir, "HIDDEN"},
		"record_size":   {props.RecordSize, "128K"},
	}
	for name, c := range stringChecks {
		if c.got == nil || *c.got != c.want {
			t.Errorf("%s: got %v, want %q", name, c.got, c.want)
		}
	}

	// Integer properties must read `parsed` (raw byte count) when present,
	// falling back to `rawvalue` for the "0"/none case (reservation/refreservation).
	intChecks := map[string]struct {
		got  *int64
		want int64
	}{
		"quota":          {props.Quota, 21474836480},
		"refquota":       {props.Refquota, 10737418240},
		"reservation":    {props.Reservation, 0},
		"refreservation": {props.Refreservation, 0},
		"copies":         {props.Copies, 1},
	}
	for name, c := range intChecks {
		if c.got == nil || *c.got != c.want {
			t.Errorf("%s: got %v, want %d", name, c.got, c.want)
		}
	}

	// *String properties mirror TrueNAS's human-formatted `value`, independent
	// of the accepts-compatible integer extraction above.
	stringSizeChecks := map[string]struct {
		got  *string
		want string
	}{
		"quota_string":    {props.QuotaString, "20 GiB"},
		"refquota_string": {props.RefquotaString, "10 GiB"},
	}
	for name, c := range stringSizeChecks {
		if c.got == nil || *c.got != c.want {
			t.Errorf("%s: got %v, want %q", name, c.got, c.want)
		}
	}
	// Reservation/refreservation report value:null for the "0"/none case
	// (see the intChecks comment above); *String stays nil there too.
	if props.ReservationString != nil {
		t.Errorf("reservation_string: got %v, want nil", props.ReservationString)
	}
	if props.RefreservationString != nil {
		t.Errorf("refreservation_string: got %v, want nil", props.RefreservationString)
	}
}

func strPtr(s string) *string { return &s }

// TestExtractPoolDatasetProperties_CommentsTopLevel verifies the top-level
// `comments` field is preferred over user_properties when both are present
// (forward-compatible with a TrueNAS version that returns it as documented).
func TestExtractPoolDatasetProperties_CommentsTopLevel(t *testing.T) {
	ds := &truenas.PoolDataset{
		Id:       "tank/fs",
		Type:     "FILESYSTEM",
		Comments: zfsProp("top-level comment", "top-level comment", json.RawMessage(`"top-level comment"`)),
		UserProperties: map[string]json.RawMessage{
			"comments": json.RawMessage(`{"value":"stale user-property comment"}`),
		},
	}
	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		t.Fatalf("ExtractPoolDatasetProperties: %v", err)
	}
	if props.Comments == nil || *props.Comments != "top-level comment" {
		t.Errorf("Comments: got %v, want %q", props.Comments, "top-level comment")
	}
}

// TestExtractPoolDatasetProperties_CommentsAbsent verifies comments is nil
// when neither the top-level field nor user_properties carries it.
func TestExtractPoolDatasetProperties_CommentsAbsent(t *testing.T) {
	ds := &truenas.PoolDataset{Id: "tank/fs", Type: "FILESYSTEM"}
	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		t.Fatalf("ExtractPoolDatasetProperties: %v", err)
	}
	if props.Comments != nil {
		t.Errorf("Comments: got %v, want nil", props.Comments)
	}
}

func TestExtractPoolDatasetProperties_NullParsed(t *testing.T) {
	ds := &truenas.PoolDataset{
		Id:    "tank/fs",
		Type:  "FILESYSTEM",
		Quota: truenas.ZFSProperty{Parsed: json.RawMessage(`null`)},
	}
	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		t.Fatalf("ExtractPoolDatasetProperties: %v", err)
	}
	if props.Quota != nil {
		t.Errorf("Quota: got %v, want nil", props.Quota)
	}
}

// TestExtractPoolDatasetProperties_NullParsedRawvalueFallback verifies that
// when `parsed` is null but `rawvalue` carries a numeric string (the
// "0"/none quirk observed for reservation/refreservation on live TrueNAS),
// extraction falls back to parsing rawvalue rather than losing the value.
func TestExtractPoolDatasetProperties_NullParsedRawvalueFallback(t *testing.T) {
	ds := &truenas.PoolDataset{
		Id:          "tank/fs",
		Type:        "FILESYSTEM",
		Reservation: truenas.ZFSProperty{RawValue: strPtr("0"), Parsed: json.RawMessage(`null`)},
	}
	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		t.Fatalf("ExtractPoolDatasetProperties: %v", err)
	}
	if props.Reservation == nil || *props.Reservation != 0 {
		t.Errorf("Reservation: got %v, want 0", props.Reservation)
	}
}

func TestExtractPoolDatasetProperties_InvalidParsed(t *testing.T) {
	ds := &truenas.PoolDataset{
		Id:    "tank/fs",
		Type:  "FILESYSTEM",
		Quota: truenas.ZFSProperty{Parsed: json.RawMessage(`"not a number"`)},
	}
	if _, err := truenas.ExtractPoolDatasetProperties(ds); err == nil {
		t.Fatal("expected error for non-numeric parsed value, got nil")
	}
}
