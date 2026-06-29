package codegen_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gringolito/terraform-provider-truenas/internal/codegen"
)

// Run with -update to regenerate golden files.
var update = flag.Bool("update", false, "update golden files")

// golden files are expected under testdata/golden/<namespace>_gen.go.
// The fixture is testdata/fixture.json, trimmed from the real api/registry.json.

func TestGenerateNamespace(t *testing.T) {
	fixtureData, err := os.ReadFile(filepath.Join("testdata", "fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v\n\nRun 'make refresh-snapshot' first, then trim testdata/fixture.json.", err)
	}

	reg, err := codegen.ParseRegistry(fixtureData)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	tests := []struct {
		ns     string
		golden string
	}{
		// These must cover the five required schema patterns:
		//  1. plain method (flat struct, single arg)
		//  2. [id, patch] update method
		//  3. polymorphic anyOf composition
		//  4. anyOf: [T, null] nullable field
		//  5. method with _required_ and _attrs_order_ annotations
		{"user", filepath.Join("testdata", "golden", "user_gen.go")},
		// group also tests:
		//  6. non-standard verb returns separate result struct (get_group_obj → GroupGetGroupObjResult)
		//  7. update args bool→*bool and slice→no omitempty
		{"group", filepath.Join("testdata", "golden", "group_gen.go")},
	}

	for _, tt := range tests {
		t.Run(tt.ns, func(t *testing.T) {
			got, err := codegen.GenerateNamespace(reg, tt.ns)
			if err != nil {
				t.Fatalf("GenerateNamespace(%q): %v", tt.ns, err)
			}

			if *update {
				if err := os.MkdirAll(filepath.Dir(tt.golden), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(tt.golden, got, 0o644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("updated %s", tt.golden)
				return
			}

			want, err := os.ReadFile(tt.golden)
			if err != nil {
				t.Fatalf("read golden %s: %v\n\nRun with -update to create it.", tt.golden, err)
			}
			if string(got) != string(want) {
				t.Errorf("golden mismatch for namespace %q\n\nRun with -update to refresh.\n\ngot:\n%s\nwant:\n%s",
					tt.ns, got, want)
			}
		})
	}
}

// TestDateTimeFieldMapsToDateTime verifies that a nullable string field carrying
// "format": "date-time" maps to *DateTime, not *string. TrueNAS serializes
// datetimes over the wire as a Mongo-style {"$date": <epoch_ms>} object, which
// cannot decode into a Go string; the DateTime type knows how to decode it.
func TestDateTimeFieldMapsToDateTime(t *testing.T) {
	fixtureData, err := os.ReadFile(filepath.Join("testdata", "fixture.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	reg, err := codegen.ParseRegistry(fixtureData)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	got, err := codegen.GenerateNamespace(reg, "user")
	if err != nil {
		t.Fatalf("GenerateNamespace(user): %v", err)
	}
	src := string(got)

	if !strings.Contains(src, "LastPasswordChange *DateTime") {
		t.Errorf("expected last_password_change to map to *DateTime; generated source:\n%s", src)
	}
	if strings.Contains(src, "LastPasswordChange *string") {
		t.Errorf("date-time field must not map to *string")
	}
}
