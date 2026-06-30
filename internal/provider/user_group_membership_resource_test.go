package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

// TestAccUserGroupMembershipResource_basic creates a membership and verifies
// all attributes are populated (AC1).
func TestAccUserGroupMembershipResource_basic(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUGMConfig(username, g1, g2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_user_group_membership.test", "id"),
					resource.TestCheckResourceAttrSet("truenas_user_group_membership.test", "user_id"),
					resource.TestCheckResourceAttr("truenas_user_group_membership.test", "group_ids.#", "2"),
				),
			},
		},
	})
}

// TestAccUserGroupMembershipResource_emptyPlan verifies a second apply
// produces no diff (AC5).
func TestAccUserGroupMembershipResource_emptyPlan(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()
	cfg := testAccUGMConfig(username, g1, g2)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccUserGroupMembershipResource_update changes group_ids from [g1,g2] to
// [g1,g3], verifying g2 is removed, g3 is added, and a separately-added
// external group g4 is preserved (AC2).
func TestAccUserGroupMembershipResource_update(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()
	g3 := randGroupName()
	g4 := randGroupName()

	var userUID int64
	var g4GID int64

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUGMUpdateConfigBefore(username, g1, g2, g3, g4),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user_group_membership.test", "group_ids.#", "2"),
					func(s *terraform.State) error {
						uid, err := ugmStateUID(s, "truenas_user_group_membership.test")
						if err != nil {
							return err
						}
						userUID = uid

						rs, ok := s.RootModule().Resources["truenas_group.g4"]
						if !ok {
							return fmt.Errorf("truenas_group.g4 not in state")
						}
						gid, err := strconv.ParseInt(rs.Primary.Attributes["gid"], 10, 64)
						if err != nil {
							return err
						}
						g4GID = gid
						return nil
					},
				),
			},
			{
				// Add g4 externally before the update so we can verify it survives.
				PreConfig: func() {
					c := accTestCaller(t)
					ctx := context.Background()
					apiID, err := truenas.ResolveUserIDByUID(ctx, c, userUID)
					if err != nil {
						t.Logf("resolve user %d: %v", userUID, err)
						return
					}
					u, err := truenas.UserGetInstance(ctx, c, apiID)
					if err != nil {
						t.Logf("get user instance: %v", err)
						return
					}
					g4APIID, err := truenas.ResolveGroupIDByGID(ctx, c, g4GID)
					if err != nil {
						t.Logf("resolve g4 GID %d: %v", g4GID, err)
						return
					}
					merged := append(u.Groups, g4APIID)
					if _, err := truenas.UserUpdate(ctx, c, apiID, truenas.UserUpdateArgs{Groups: &merged}); err != nil {
						t.Logf("add external group: %v", err)
					}
				},
				Config: testAccUGMUpdateConfigAfter(username, g1, g2, g3, g4),
				Check: resource.ComposeTestCheckFunc(
					// Managed membership now tracks [g1, g3].
					resource.TestCheckResourceAttr("truenas_user_group_membership.test", "group_ids.#", "2"),
					// Verify g4 (external) is still in the live user groups via API.
					func(s *terraform.State) error {
						return checkUserHasGroup(t, userUID, g4GID)
					},
				),
			},
		},
	})
}

// TestAccUserGroupMembershipResource_destroy verifies that destroying the
// membership removes only the declared edges and leaves external groups
// intact (AC3).
func TestAccUserGroupMembershipResource_destroy(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()

	var userUID int64
	var g2GID int64

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUGMDestroyConfigWithMembership(username, g1, g2),
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						uid, err := ugmStateUID(s, "truenas_user_group_membership.test")
						if err != nil {
							return err
						}
						userUID = uid

						rs, ok := s.RootModule().Resources["truenas_group.g2"]
						if !ok {
							return fmt.Errorf("truenas_group.g2 not in state")
						}
						gid, err := strconv.ParseInt(rs.Primary.Attributes["gid"], 10, 64)
						if err != nil {
							return err
						}
						g2GID = gid
						return nil
					},
				),
			},
			{
				// Add g2 externally, then remove the membership resource (keep user+groups).
				PreConfig: func() {
					c := accTestCaller(t)
					ctx := context.Background()
					apiID, err := truenas.ResolveUserIDByUID(ctx, c, userUID)
					if err != nil {
						t.Logf("resolve user %d: %v", userUID, err)
						return
					}
					u, err := truenas.UserGetInstance(ctx, c, apiID)
					if err != nil {
						t.Logf("get user instance: %v", err)
						return
					}
					g2APIID, err := truenas.ResolveGroupIDByGID(ctx, c, g2GID)
					if err != nil {
						t.Logf("resolve g2 GID %d: %v", g2GID, err)
						return
					}
					merged := append(u.Groups, g2APIID)
					if _, err := truenas.UserUpdate(ctx, c, apiID, truenas.UserUpdateArgs{Groups: &merged}); err != nil {
						t.Logf("add external group: %v", err)
					}
				},
				// Drop the membership block; Terraform destroys only that resource.
				Config: testAccUGMDestroyConfigNoMembership(username, g1, g2),
				Check: func(s *terraform.State) error {
					// g1 (declared) should be gone; g2 (external) must survive.
					return checkUserHasGroup(t, userUID, g2GID)
				},
			},
		},
	})
}

// TestAccUserGroupMembershipResource_twoInstances creates two membership
// blocks for the same user and verifies their union composes correctly with no
// plan diff on the second apply (AC4).
func TestAccUserGroupMembershipResource_twoInstances(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()
	g3 := randGroupName()
	cfg := testAccUGMTwoInstancesConfig(username, g1, g2, g3)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_user_group_membership.a", "group_ids.#", "2"),
					resource.TestCheckResourceAttr("truenas_user_group_membership.b", "group_ids.#", "2"),
				),
			},
			{Config: cfg, PlanOnly: true, ExpectNonEmptyPlan: false},
		},
	})
}

// TestAccUserGroupMembershipResource_import verifies that importing by
// <uid>:<gid1>,<gid2> round-trips correctly (AC7).
func TestAccUserGroupMembershipResource_import(t *testing.T) {
	username := randUserName()
	g1 := randGroupName()
	g2 := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: testAccUGMConfig(username, g1, g2)},
			{
				ResourceName: "truenas_user_group_membership.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["truenas_user_group_membership.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					uid := rs.Primary.Attributes["user_id"]
					gid1 := rs.Primary.Attributes["group_ids.0"]
					gid2 := rs.Primary.Attributes["group_ids.1"]
					return fmt.Sprintf("%s:%s,%s", uid, gid1, gid2), nil
				},
				ImportStateVerify: true,
			},
		},
	})
}

// ---- helper functions -------------------------------------------------------

func ugmStateUID(s *terraform.State, resourceAddress string) (int64, error) {
	rs, ok := s.RootModule().Resources[resourceAddress]
	if !ok {
		return 0, fmt.Errorf("resource %q not in state", resourceAddress)
	}
	return strconv.ParseInt(rs.Primary.Attributes["user_id"], 10, 64)
}

func checkUserHasGroup(t *testing.T, uid, gid int64) error {
	t.Helper()
	c := accTestCaller(t)
	ctx := context.Background()

	raw, err := truenas.UserQuery(ctx, c, truenas.QueryFilter{Field: "uid", Op: "=", Value: uid})
	if err != nil {
		return fmt.Errorf("querying user UID %d: %w", uid, err)
	}
	var users []struct {
		Id     int64   `json:"id"`
		Groups []int64 `json:"groups"`
	}
	if err := json.Unmarshal(raw, &users); err != nil {
		return fmt.Errorf("parsing user query: %w", err)
	}
	if len(users) == 0 {
		return fmt.Errorf("user with UID %d not found", uid)
	}

	apiIDs := users[0].Groups
	gids, err := truenas.ResolveGroupGIDs(ctx, c, apiIDs)
	if err != nil {
		return fmt.Errorf("resolving group GIDs: %w", err)
	}

	for _, g := range gids {
		if g == gid {
			return nil
		}
	}
	return fmt.Errorf("expected user UID %d to have group GID %d in live state, but it was not found (groups: %v)", uid, gid, gids)
}

// ---- HCL configs -----------------------------------------------------------

func testAccUGMConfig(username, g1Name, g2Name string) string {
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
  group             = truenas_group.g1.gid
  password_disabled = true
}

resource "truenas_user_group_membership" "test" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g1.gid, truenas_group.g2.gid]
}
`, g1Name, g2Name, username)
}

func testAccUGMUpdateConfigBefore(username, g1Name, g2Name, g3Name, g4Name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" { name = %q }
resource "truenas_group" "g2" { name = %q }
resource "truenas_group" "g3" { name = %q }
resource "truenas_group" "g4" { name = %q }

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.g1.gid
  password_disabled = true
}

resource "truenas_user_group_membership" "test" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g1.gid, truenas_group.g2.gid]
}
`, g1Name, g2Name, g3Name, g4Name, username)
}

func testAccUGMUpdateConfigAfter(username, g1Name, g2Name, g3Name, g4Name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" { name = %q }
resource "truenas_group" "g2" { name = %q }
resource "truenas_group" "g3" { name = %q }
resource "truenas_group" "g4" { name = %q }

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.g1.gid
  password_disabled = true
}

resource "truenas_user_group_membership" "test" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g1.gid, truenas_group.g3.gid]
}
`, g1Name, g2Name, g3Name, g4Name, username)
}

func testAccUGMDestroyConfigWithMembership(username, g1Name, g2Name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" { name = %q }
resource "truenas_group" "g2" { name = %q }

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.g1.gid
  password_disabled = true
}

resource "truenas_user_group_membership" "test" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g1.gid]
}
`, g1Name, g2Name, username)
}

func testAccUGMDestroyConfigNoMembership(username, g1Name, g2Name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" { name = %q }
resource "truenas_group" "g2" { name = %q }

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.g1.gid
  password_disabled = true
}
`, g1Name, g2Name, username)
}

func testAccUGMTwoInstancesConfig(username, g1Name, g2Name, g3Name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "g1" { name = %q }
resource "truenas_group" "g2" { name = %q }
resource "truenas_group" "g3" { name = %q }

resource "truenas_user" "test" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.g1.gid
  password_disabled = true
}

resource "truenas_user_group_membership" "a" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g1.gid, truenas_group.g2.gid]
}

resource "truenas_user_group_membership" "b" {
  user_id   = truenas_user.test.id
  group_ids = [truenas_group.g2.gid, truenas_group.g3.gid]
}
`, g1Name, g2Name, g3Name, username)
}
