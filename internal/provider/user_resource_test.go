package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

func randUserName() string {
	return fmt.Sprintf("tf-acc-%d", rand.Intn(900000)+100000)
}

// TestAccUserResource_basic creates a user and verifies all attributes are populated.
func TestAccUserResource_basic(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(groupName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "username", name),
					resource.TestCheckResourceAttrSet("truenas_user.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "uid"),
					resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Test User"),
					resource.TestCheckResourceAttr("truenas_user.test", "local", "true"),
					resource.TestCheckResourceAttr("truenas_user.test", "builtin", "false"),
					resource.TestCheckResourceAttrSet("truenas_user.test", "group"),
				),
			},
		},
	})
}

// TestAccUserResource_emptyPlan verifies that a second apply produces no diff.
func TestAccUserResource_emptyPlan(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	cfg := testAccUserConfig(groupName, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccUserResource_update changes the full_name and verifies in-place update.
func TestAccUserResource_update(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(groupName, name),
				Check:  resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Test User"),
			},
			{
				Config: testAccUserConfigUpdated(groupName, name),
				Check:  resource.TestCheckResourceAttr("truenas_user.test", "full_name", "Updated User"),
			},
		},
	})
}

// TestAccUserResource_import verifies importing by UID round-trips correctly.
func TestAccUserResource_import(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccUserConfig(groupName, name)},
			{
				ResourceName:            "truenas_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password_wo_version", "home_create", "home_mode"},
			},
		},
	})
}

// TestAccUserResource_passwordWriteOnly verifies password is never in state and
// that bumping the version counter triggers a password update.
func TestAccUserResource_passwordWriteOnly(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigWithPassword(groupName, name, "s3cr3t!", 1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("truenas_user.test", "password"),
					resource.TestCheckResourceAttr("truenas_user.test", "password_wo_version", "1"),
				),
			},
			{
				// Bump version → provider resends password.
				Config: testAccUserConfigWithPassword(groupName, name, "n3wp4ss!", 2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr("truenas_user.test", "password"),
					resource.TestCheckResourceAttr("truenas_user.test", "password_wo_version", "2"),
				),
			},
		},
	})
}

// TestAccUserResource_passwordDisabled creates a user with password_disabled=true.
func TestAccUserResource_passwordDisabled(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigPasswordDisabled(groupName, name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user.test", "password_disabled", "true"),
					resource.TestCheckNoResourceAttr("truenas_user.test", "password"),
				),
			},
		},
	})
}

// TestAccUserResource_groupChange changes the primary group without replacing the resource.
func TestAccUserResource_groupChange(t *testing.T) {
	name := randUserName()
	groupName1 := randGroupName()
	groupName2 := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfigTwoGroups(groupName1, groupName2, name, "truenas_group.g1.gid"),
				Check:  resource.TestCheckResourceAttrSet("truenas_user.test", "group"),
			},
			{
				Config: testAccUserConfigTwoGroups(groupName1, groupName2, name, "truenas_group.g2.gid"),
				Check:  resource.TestCheckResourceAttrSet("truenas_user.test", "group"),
			},
		},
	})
}

// TestAccUserResource_groupsReadOnly verifies that supplementary groups never cause a non-empty plan.
func TestAccUserResource_groupsReadOnly(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	cfg := testAccUserConfig(groupName, name)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccUserResource_outOfBandDeletion deletes the user externally and verifies
// that Terraform detects the drift without error.
func TestAccUserResource_outOfBandDeletion(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	var userUID int64
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig(groupName, name),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_user.test"]
						if !ok {
							return fmt.Errorf("resource not in state")
						}
						uid, err := strconv.ParseInt(rs.Primary.Attributes["id"], 10, 64)
						if err != nil {
							return err
						}
						userUID = uid
						return nil
					},
				),
			},
			{
				PreConfig: func() {
					c := accTestCaller(t)
					raw, err := truenas.UserQuery(context.Background(), c,
						truenas.QueryFilter{Field: "uid", Op: "=", Value: userUID})
					if err != nil {
						t.Logf("out-of-band query: %v", err)
						return
					}
					var users []struct {
						Id int64 `json:"id"`
					}
					if err := json.Unmarshal(raw, &users); err != nil {
						t.Logf("out-of-band parse: %v", err)
						return
					}
					if len(users) == 0 {
						t.Logf("out-of-band: user UID %d not found", userUID)
						return
					}
					if err := truenas.UserDelete(context.Background(), c, users[0].Id); err != nil {
						t.Logf("out-of-band delete: %v (may be ok if already gone)", err)
					}
				},
				Config:             testAccUserConfig(groupName, name),
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// ---- HCL configs -----------------------------------------------------------

func testAccUserConfig(groupName, username string) string {
	return fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
}

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.test.gid
  password_disabled = true
}
`, groupName, username)
}

func testAccUserConfigUpdated(groupName, username string) string {
	return fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
}

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Updated User"
  group             = truenas_group.test.gid
  password_disabled = true
}
`, groupName, username)
}

func testAccUserConfigWithPassword(groupName, username, password string, version int) string {
	return fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
}

resource "truenas_user" "test" {
  username            = %q
  full_name           = "Test User"
  group               = truenas_group.test.gid
  password            = %q
  password_wo_version = %d
}
`, groupName, username, password, version)
}

func testAccUserConfigPasswordDisabled(groupName, username string) string {
	return fmt.Sprintf(`
resource "truenas_group" "test" {
  name = %q
}

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.test.gid
  password_disabled = true
}
`, groupName, username)
}

func testAccUserConfigTwoGroups(groupName1, groupName2, username, groupRef string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" {
  name = %q
}

resource "truenas_group" "g2" {
  name = %q
}

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = %s
  password_disabled = true
}
`, groupName1, groupName2, username, groupRef)
}
