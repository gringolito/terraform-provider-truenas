package provider_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPoolDataSource_byName(t *testing.T) {
	poolName := os.Getenv("TRUENAS_TEST_POOL")
	if poolName == "" {
		t.Skip("TRUENAS_TEST_POOL not set")
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPoolDataSourceByNameConfig(poolName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_pool.test", "name", poolName),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "id"),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "status"),
					resource.TestCheckResourceAttrSet("data.truenas_pool.test", "healthy"),
				),
			},
		},
	})
}

func TestAccPoolDataSource_notFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccPoolDataSourceByNameConfig("this-pool-does-not-exist-tf-acc"),
				ExpectError: regexp.MustCompile(`no pool with name`),
			},
		},
	})
}

func testAccPoolDataSourceByNameConfig(name string) string {
	return fmt.Sprintf(`
data "truenas_pool" "test" {
  name = %q
}
`, name)
}
