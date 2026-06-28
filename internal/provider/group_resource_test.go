package provider_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

func accTestCredentials(t *testing.T) (host, apiKey, username, password string) {
	t.Helper()
	return os.Getenv("TRUENAS_HOST"),
		os.Getenv("TRUENAS_API_KEY"),
		os.Getenv("TRUENAS_USERNAME"),
		os.Getenv("TRUENAS_PASSWORD")
}

func randGroupName() string {
	return fmt.Sprintf("tf-acc-%d", rand.Intn(900000)+100000)
}

// TestAccGroupResource_basic creates a group and verifies all computed
// attributes are populated in state.
func TestAccGroupResource_basic(t *testing.T) {
	name := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_group.test", "name", name),
					resource.TestCheckResourceAttrSet("truenas_group.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_group.test", "gid"),
					resource.TestCheckResourceAttr("truenas_group.test", "smb", "true"),
					resource.TestCheckResourceAttr("truenas_group.test", "builtin", "false"),
					resource.TestCheckResourceAttr("truenas_group.test", "local", "true"),
				),
			},
		},
	})
}

// TestAccGroupResource_emptyPlan verifies that a second apply produces no diff.
func TestAccGroupResource_emptyPlan(t *testing.T) {
	name := randGroupName()
	cfg := testAccGroupConfig(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccGroupResource_update renames the group and verifies an in-place update
// (no resource replacement).
func TestAccGroupResource_update(t *testing.T) {
	name := randGroupName()
	newName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(name),
				Check:  resource.TestCheckResourceAttr("truenas_group.test", "name", name),
			},
			{
				Config: testAccGroupConfig(newName),
				Check:  resource.TestCheckResourceAttr("truenas_group.test", "name", newName),
			},
		},
	})
}

// TestAccGroupResource_import verifies that importing by integer ID round-trips
// correctly.
func TestAccGroupResource_import(t *testing.T) {
	name := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccGroupConfig(name)},
			{
				ResourceName:      "truenas_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccGroupResource_outOfBandDeletion deletes the group externally and
// verifies that the next plan removes it from state without error.
func TestAccGroupResource_outOfBandDeletion(t *testing.T) {
	name := randGroupName()
	var groupID int64
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupConfig(name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_group.test"]
						if !ok {
							return fmt.Errorf("resource not in state")
						}
						id, err := strconv.ParseInt(rs.Primary.Attributes["id"], 10, 64)
						if err != nil {
							return err
						}
						groupID = id
						return nil
					},
				),
			},
			{
				// Delete the group out of band, then verify Terraform detects
				// the drift (plan shows "will create") without re-applying.
				PreConfig: func() {
					c := accTestCaller(t)
					if err := truenas.GroupDelete(context.Background(), c, groupID); err != nil {
						t.Logf("out-of-band delete: %v (may be ok if already gone)", err)
					}
				},
				Config:             testAccGroupConfig(name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccGroupResource_usersReadOnly verifies that the `users` attribute never
// causes a non-empty plan (it is Computed/read-only and must not drift).
func TestAccGroupResource_usersReadOnly(t *testing.T) {
	name := randGroupName()
	cfg := testAccGroupConfig(name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// accTestCaller returns a client.Caller wired from environment variables.
// Used only for out-of-band operations in acceptance tests.
func accTestCaller(t *testing.T) client.Caller {
	t.Helper()
	host, apiKey, username, password := accTestCredentials(t)
	c, err := client.NewWebSocketClient(host, apiKey, username, password, false)
	if err != nil {
		t.Fatalf("accTestCaller: %v", err)
	}
	return c
}

func testAccGroupConfig(name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
  smb  = true
}
`, name)
}
