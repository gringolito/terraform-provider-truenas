package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccGroupDataSource_byId looks up a group using the integer id attribute.
func TestAccGroupDataSource_byId(t *testing.T) {
	name := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceByIdConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_group.by_id", "name", name),
					resource.TestCheckResourceAttrSet("data.truenas_group.by_id", "gid"),
					resource.TestCheckResourceAttr("data.truenas_group.by_id", "local", "true"),
				),
			},
		},
	})
}

// TestAccGroupDataSource_byName looks up a group using the name attribute.
func TestAccGroupDataSource_byName(t *testing.T) {
	name := randGroupName()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceByNameConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.truenas_group.by_name", "id"),
					resource.TestCheckResourceAttr("data.truenas_group.by_name", "name", name),
					resource.TestCheckResourceAttr("data.truenas_group.by_name", "local", "true"),
				),
			},
		},
	})
}

func testAccGroupDataSourceByIdConfig(name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "seed" {
  name = %q
}

data "truenas_group" "by_id" {
  id = truenas_group.seed.id
}
`, name)
}

func testAccGroupDataSourceByNameConfig(name string) string {
	return fmt.Sprintf(`
resource "truenas_group" "seed" {
  name = %q
}

data "truenas_group" "by_name" {
  name = truenas_group.seed.name
}
`, name)
}
