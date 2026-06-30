package truenas_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gringolito/terraform-provider-truenas/internal/client/clienttest"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

func TestPoolGetByName(t *testing.T) {
	payload := `[{
		"id": 1,
		"name": "tank",
		"guid": "6122501628162107165",
		"status": "ONLINE",
		"path": "/mnt/tank",
		"healthy": true,
		"warning": false,
		"status_code": "OK",
		"status_detail": null,
		"size": 105763569664,
		"allocated": 19956891648,
		"free": 85806678016,
		"freeing": 0,
		"dedup_table_size": 0,
		"dedup_table_quota": "auto",
		"fragmentation": "33",
		"autotrim": {"value": "off", "rawvalue": "off", "parsed": "off", "source": "DEFAULT"},
		"scan": null
	}]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.query": json.RawMessage(payload),
		},
	}
	p, err := truenas.PoolGetByName(context.Background(), fake, "tank")
	if err != nil {
		t.Fatalf("PoolGetByName: %v", err)
	}
	if p.Id != 1 {
		t.Errorf("Id: got %d, want 1", p.Id)
	}
	if p.Name != "tank" {
		t.Errorf("Name: got %q, want %q", p.Name, "tank")
	}
	if !p.Healthy {
		t.Error("Healthy: got false, want true")
	}
	if p.Warning {
		t.Error("Warning: got true, want false")
	}
	if p.Autotrim.Parsed != "off" {
		t.Errorf("Autotrim.Parsed: got %q, want %q", p.Autotrim.Parsed, "off")
	}
	if p.Scan != nil {
		t.Errorf("Scan: got %v, want nil", p.Scan)
	}
}

func TestPoolGetByName_NotFound(t *testing.T) {
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.query": json.RawMessage(`[]`),
		},
	}
	_, err := truenas.PoolGetByName(context.Background(), fake, "missing")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPoolGetByName_WithScan(t *testing.T) {
	payload := `[{
		"id": 1,
		"name": "Man",
		"guid": "6122501628162107165",
		"status": "ONLINE",
		"path": "/mnt/Man",
		"healthy": true,
		"warning": false,
		"status_code": "OK",
		"status_detail": null,
		"size": 105763569664,
		"allocated": 19956891648,
		"free": 85806678016,
		"freeing": 0,
		"dedup_table_size": 0,
		"dedup_table_quota": "auto",
		"fragmentation": "33",
		"autotrim": {"value": "off", "rawvalue": "off", "parsed": "off", "source": "DEFAULT"},
		"scan": {
			"function": "SCRUB",
			"state": "FINISHED",
			"start_time": {"$date": 1781406001000},
			"end_time": {"$date": 1781406168000},
			"percentage": 99.9933660030365,
			"bytes_to_process": 19755823104,
			"bytes_processed": 19757133824,
			"bytes_issued": 19750625280,
			"pause": null,
			"errors": 0,
			"total_secs_left": null
		}
	}]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.query": json.RawMessage(payload),
		},
	}
	p, err := truenas.PoolGetByName(context.Background(), fake, "Man")
	if err != nil {
		t.Fatalf("PoolGetByName: %v", err)
	}
	if p.Scan == nil {
		t.Fatal("Scan: got nil, want non-nil")
	}
	if p.Scan.Function != "SCRUB" {
		t.Errorf("Scan.Function: got %q, want %q", p.Scan.Function, "SCRUB")
	}
	if p.Scan.State != "FINISHED" {
		t.Errorf("Scan.State: got %q, want %q", p.Scan.State, "FINISHED")
	}
	if p.Scan.StartTime.IsZero() {
		t.Error("Scan.StartTime: got zero, want non-zero")
	}
	if p.Scan.EndTime.IsZero() {
		t.Error("Scan.EndTime: got zero, want non-zero")
	}
	if p.Scan.Percentage != 99.9933660030365 {
		t.Errorf("Scan.Percentage: got %f, want 99.9933660030365", p.Scan.Percentage)
	}
	if p.Scan.Errors != 0 {
		t.Errorf("Scan.Errors: got %d, want 0", p.Scan.Errors)
	}
	if p.Scan.Pause != nil {
		t.Errorf("Scan.Pause: got %v, want nil", p.Scan.Pause)
	}
	if p.Scan.TotalSecsLeft != nil {
		t.Errorf("Scan.TotalSecsLeft: got %v, want nil", p.Scan.TotalSecsLeft)
	}
}

func TestPoolGetByName_ScanNull(t *testing.T) {
	payload := `[{
		"id": 2,
		"name": "newpool",
		"guid": "1234567890",
		"status": "ONLINE",
		"path": "/mnt/newpool",
		"healthy": true,
		"warning": false,
		"status_code": null,
		"status_detail": null,
		"size": null,
		"allocated": null,
		"free": null,
		"freeing": null,
		"dedup_table_size": null,
		"dedup_table_quota": null,
		"fragmentation": null,
		"autotrim": {"value": "off", "rawvalue": "off", "parsed": "off", "source": "DEFAULT"},
		"scan": null
	}]`
	fake := &clienttest.FakeCaller{
		Responses: map[string]json.RawMessage{
			"pool.query": json.RawMessage(payload),
		},
	}
	p, err := truenas.PoolGetByName(context.Background(), fake, "newpool")
	if err != nil {
		t.Fatalf("PoolGetByName: %v", err)
	}
	if p.Scan != nil {
		t.Errorf("Scan: got %v, want nil", p.Scan)
	}
	if p.Size != nil {
		t.Errorf("Size: got %v, want nil", p.Size)
	}
	if p.StatusCode != nil {
		t.Errorf("StatusCode: got %v, want nil", p.StatusCode)
	}
}
