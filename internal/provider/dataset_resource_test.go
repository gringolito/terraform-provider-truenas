package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

func init() {
	resource.AddTestSweepers("truenas_dataset", &resource.Sweeper{
		Name: "truenas_dataset",
		F:    sweepDatasets,
	})
}

// sweepDatasets deletes any leftover <pool>/tf-acc-tests/tf-acc-* datasets
// left behind by interrupted or failed acceptance test runs. The parent
// dataset itself is never swept. This provider has no region concept, so the
// sweeper argument is ignored.
func sweepDatasets(_ string) error {
	pool := os.Getenv("TRUENAS_TEST_POOL")
	if pool == "" || os.Getenv("TRUENAS_HOST") == "" {
		return nil
	}
	insecureSkipVerify, _ := strconv.ParseBool(os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY"))
	c, err := client.NewWebSocketClient(
		os.Getenv("TRUENAS_HOST"), os.Getenv("TRUENAS_API_KEY"),
		os.Getenv("TRUENAS_USERNAME"), os.Getenv("TRUENAS_PASSWORD"), insecureSkipVerify)
	if err != nil {
		return fmt.Errorf("sweeper: connect: %w", err)
	}

	ctx := context.Background()
	prefix := datasetTestParentPath(pool) + "/tf-acc-"
	raw, err := truenas.PoolDatasetQuery(ctx, c, truenas.QueryFilter{Field: "id", Op: "~", Value: "^" + regexp.QuoteMeta(prefix)})
	if err != nil {
		return fmt.Errorf("sweeper: querying leftover datasets: %w", err)
	}
	var datasets []struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(raw, &datasets); err != nil {
		return fmt.Errorf("sweeper: parsing query result: %w", err)
	}

	var errs []string
	for _, ds := range datasets {
		if err := truenas.PoolDatasetDelete(ctx, c, ds.Id); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", ds.Id, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sweeper: failed to delete %d dataset(s): %s", len(errs), strings.Join(errs, "; "))
	}
	return nil
}

func randDatasetName() string {
	return fmt.Sprintf("tf-acc-%d", rand.Intn(900000)+100000)
}

func datasetTestParentPath(pool string) string {
	return pool + "/tf-acc-tests"
}

// randDatasetPath returns a fresh ephemeral dataset path under
// <TRUENAS_TEST_POOL>/tf-acc-tests. Safe to call unconditionally (no network
// I/O): TRUENAS_TEST_POOL may be empty here when TF_ACC is unset, since the
// resulting string is never handed to Terraform unless testAccDatasetPreCheck
// (which gates on TF_ACC and skips otherwise) has already run.
func randDatasetPath() string {
	return datasetTestParentPath(os.Getenv("TRUENAS_TEST_POOL")) + "/" + randDatasetName()
}

// testAccDatasetPreCheck skips the test unless TRUENAS_TEST_POOL is set (in
// addition to the base TF_ACC/TRUENAS_HOST gating), then ensures
// <pool>/tf-acc-tests exists as the dedicated parent for ephemeral test
// datasets. The parent is created once and never deleted by the tests.
func testAccDatasetPreCheck(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	pool := os.Getenv("TRUENAS_TEST_POOL")
	if pool == "" {
		t.Skip("TRUENAS_TEST_POOL not set")
	}

	c := accTestCaller(t)
	parent := datasetTestParentPath(pool)
	if _, err := truenas.PoolDatasetGetInstance(context.Background(), c, parent); err == nil {
		return
	}
	if _, err := truenas.PoolDatasetCreate(context.Background(), c, truenas.PoolDatasetCreateArgs{Name: parent}); err != nil {
		t.Fatalf("ensure parent dataset %q: %v", parent, err)
	}
}

// TestAccDatasetResource_basic creates a dataset and verifies its computed
// attributes, including the pool/name split derived from path.
func TestAccDatasetResource_basic(t *testing.T) {
	path := randDatasetPath()
	pool, _, _ := strings.Cut(path, "/")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigBasic(path),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_dataset.test", "path", path),
					resource.TestCheckResourceAttr("truenas_dataset.test", "id", path),
					resource.TestCheckResourceAttr("truenas_dataset.test", "pool", pool),
					resource.TestCheckResourceAttrSet("truenas_dataset.test", "name"),
					resource.TestCheckResourceAttr("truenas_dataset.test", "prevent_deletion", "false"),
					resource.TestCheckResourceAttr("truenas_dataset.test", "comments", "tf-acc test dataset"),
					resource.TestCheckResourceAttr("truenas_dataset.test", "compression", "LZ4"),
					resource.TestCheckResourceAttrSet("truenas_dataset.test", "atime"),
					resource.TestCheckResourceAttrSet("truenas_dataset.test", "record_size"),
				),
			},
		},
	})
}

// TestAccDatasetResource_emptyPlan verifies that a second apply produces no
// diff, accounting for the ZFS {value,parsed,source} response shape on every
// supported property (Acceptance Criteria: readable via `terraform plan` with
// no diff after apply).
func TestAccDatasetResource_emptyPlan(t *testing.T) {
	path := randDatasetPath()
	cfg := testAccDatasetConfigFull(path)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccDatasetResource_update changes compression in place and verifies the
// plan action is Update, not Replace.
func TestAccDatasetResource_update(t *testing.T) {
	path := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigCompression(path, "LZ4"),
				Check:  resource.TestCheckResourceAttr("truenas_dataset.test", "compression", "LZ4"),
			},
			{
				Config: testAccDatasetConfigCompression(path, "ZSTD"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_dataset.test", "compression", "ZSTD"),
			},
		},
	})
}

// TestAccDatasetResource_pathForceNew verifies that changing path triggers a
// ForceNew replacement (Acceptance Criteria: changing name/path forces
// replacement).
func TestAccDatasetResource_pathForceNew(t *testing.T) {
	path1 := randDatasetPath()
	path2 := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccDatasetConfigBasic(path1)},
			{
				Config: testAccDatasetConfigBasic(path2),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("truenas_dataset.test", plancheck.ResourceActionReplace),
					},
				},
				Check: resource.TestCheckResourceAttr("truenas_dataset.test", "path", path2),
			},
		},
	})
}

// TestAccDatasetResource_preventDeletionBlocksDelete verifies that
// prevent_deletion=true blocks an explicit delete, and that flipping it back
// to false allows normal deletion.
func TestAccDatasetResource_preventDeletionBlocksDelete(t *testing.T) {
	path := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigPreventDeletion(path, true),
				Check:  resource.TestCheckResourceAttr("truenas_dataset.test", "prevent_deletion", "true"),
			},
			{
				// Removing the resource from config plans a destroy, which must fail.
				Config:      "# no resources\n",
				ExpectError: regexp.MustCompile(`(?i)prevent_deletion`),
			},
			{
				// Flip back to false so the framework's own end-of-test cleanup can destroy it.
				Config: testAccDatasetConfigPreventDeletion(path, false),
				Check:  resource.TestCheckResourceAttr("truenas_dataset.test", "prevent_deletion", "false"),
			},
		},
	})
}

// TestAccDatasetResource_preventDeletionBlocksForceNew verifies that
// prevent_deletion=true also blocks a ForceNew-triggered replacement, since
// Delete runs first in that cycle (docs/adr/0003).
func TestAccDatasetResource_preventDeletionBlocksForceNew(t *testing.T) {
	path1 := randDatasetPath()
	path2 := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetConfigPreventDeletion(path1, true),
				Check:  resource.TestCheckResourceAttr("truenas_dataset.test", "prevent_deletion", "true"),
			},
			{
				// path change forces replace; Delete(old) runs first and must fail.
				Config:      testAccDatasetConfigPreventDeletion(path2, true),
				ExpectError: regexp.MustCompile(`(?i)prevent_deletion`),
			},
			{
				// Flip back to false at the original path so end-of-test cleanup succeeds.
				Config: testAccDatasetConfigPreventDeletion(path1, false),
				Check:  resource.TestCheckResourceAttr("truenas_dataset.test", "prevent_deletion", "false"),
			},
		},
	})
}

// TestAccDatasetResource_import verifies that importing by path round-trips
// correctly with an empty plan afterward.
func TestAccDatasetResource_import(t *testing.T) {
	path := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccDatasetConfigFull(path)},
			{
				ResourceName:      "truenas_dataset.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccDatasetResource_volumeTypeImportError verifies that attempting to
// import a VOLUME-type dataset with this resource returns a clear, actionable
// error. The VOLUME dataset is created out of band (this resource cannot
// create one).
func TestAccDatasetResource_volumeTypeImportError(t *testing.T) {
	path := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDatasetPreCheck(t)
			createVolumeDataset(t, path)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:        testAccDatasetConfigBasic(path),
				ResourceName:  "truenas_dataset.test",
				ImportState:   true,
				ImportStateId: path,
				ExpectError:   regexp.MustCompile(`(?i)FILESYSTEM`),
			},
		},
	})
}

// createVolumeDataset creates a throwaway VOLUME dataset directly via the raw
// API (this provider's typed client only supports FILESYSTEM create, per
// docs/adr/0007) and registers its cleanup.
func createVolumeDataset(t *testing.T, path string) {
	t.Helper()
	c := accTestCaller(t)
	_, err := c.Call(context.Background(), "pool.dataset.create", []any{map[string]any{
		"name":    path,
		"type":    "VOLUME",
		"volsize": 1073741824,
	}})
	if err != nil {
		t.Fatalf("creating throwaway VOLUME dataset %q: %v", path, err)
	}
	t.Cleanup(func() {
		_ = truenas.PoolDatasetDelete(context.Background(), c, path)
	})
}

func testAccDatasetConfigBasic(path string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "test" {
  path        = %q
  comments    = "tf-acc test dataset"
  compression = "LZ4"
}
`, path)
}

func testAccDatasetConfigCompression(path, compression string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "test" {
  path        = %q
  compression = %q
}
`, path, compression)
}

func testAccDatasetConfigPreventDeletion(path string, preventDeletion bool) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "test" {
  path             = %q
  prevent_deletion = %t
}
`, path, preventDeletion)
}

// testAccDatasetConfigFull sets every mutable property so empty-plan and
// import-round-trip tests exercise the full value/parsed extraction table.
func testAccDatasetConfigFull(path string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "test" {
  path           = %q
  comments       = "tf-acc full config"
  compression    = "ZSTD"
  sync           = "STANDARD"
  atime          = "OFF"
  exec           = "ON"
  readonly       = "OFF"
  deduplication  = "OFF"
  snap_dir       = "HIDDEN"
  record_size    = "128K"
  quota          = 2147483648
  refquota       = 2147483648
  reservation    = 0
  refreservation = 0
  copies         = 1
}
`, path)
}
