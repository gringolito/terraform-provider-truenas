package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserDataSource_byId reads a user via the id (Unix UID) attribute.
func TestAccUserDataSource_byId(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceByIdConfig(groupName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_user.by_id", "username", name),
					resource.TestCheckResourceAttr("data.truenas_user.by_id", "full_name", "Test User"),
					resource.TestCheckResourceAttr("data.truenas_user.by_id", "local", "true"),
				),
			},
		},
	})
}

// TestAccUserDataSource_byUsername reads a user via the username attribute.
func TestAccUserDataSource_byUsername(t *testing.T) {
	name := randUserName()
	groupName := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserDataSourceByUsernameConfig(groupName, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_user.by_username", "id"),
					resource.TestCheckResourceAttr("data.truenas_user.by_username", "username", name),
					resource.TestCheckResourceAttr("data.truenas_user.by_username", "local", "true"),
				),
			},
		},
	})
}

// TestAccUserDataSource_notFound verifies a useful error when the user does not exist.
func TestAccUserDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      `data "truenas_user" "missing" { username = "this-user-does-not-exist-tf-acc" }`,
				ExpectError: regexp.MustCompile(`no user with username`),
			},
		},
	})
}

// ---- HCL configs -----------------------------------------------------------

func testAccUserDataSourceByIdConfig(groupName, username string) string {
	return fmt.Sprintf(`
resource "truenas_group" "seed" {
  name = %q
}

resource "truenas_user" "seed" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.seed.gid
  password_disabled = true
}

data "truenas_user" "by_id" {
  id = truenas_user.seed.id
}
`, groupName, username)
}

func testAccUserDataSourceByUsernameConfig(groupName, username string) string {
	return fmt.Sprintf(`
resource "truenas_group" "seed" {
  name = %q
}

resource "truenas_user" "seed" {
  username          = %q
  full_name         = "Test User"
  group             = truenas_group.seed.gid
  password_disabled = true
}

data "truenas_user" "by_username" {
  username = truenas_user.seed.username
}
`, groupName, username)
}
