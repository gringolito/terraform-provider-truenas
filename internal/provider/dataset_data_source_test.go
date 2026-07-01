package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDatasetDataSource_byPath creates a dataset via the resource, then
// looks it up by path via the data source and verifies attributes match.
func TestAccDatasetDataSource_byPath(t *testing.T) {
	path := randDatasetPath()
	pool, _, _ := strings.Cut(path, "/")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccDatasetPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDatasetDataSourceConfig(path),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.truenas_dataset.test", "path", path),
					resource.TestCheckResourceAttr("data.truenas_dataset.test", "id", path),
					resource.TestCheckResourceAttr("data.truenas_dataset.test", "pool", pool),
					resource.TestCheckResourceAttrSet("data.truenas_dataset.test", "name"),
					resource.TestCheckResourceAttr("data.truenas_dataset.test", "comments", "tf-acc datasource test"),
					resource.TestCheckResourceAttr("data.truenas_dataset.test", "compression", "LZ4"),
				),
			},
		},
	})
}

// TestAccDatasetDataSource_volumeTypeError verifies that looking up a
// VOLUME-type dataset via the data source returns a clear, actionable error.
func TestAccDatasetDataSource_volumeTypeError(t *testing.T) {
	path := randDatasetPath()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDatasetPreCheck(t)
			createVolumeDataset(t, path)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDatasetDataSourceByPathOnlyConfig(path),
				ExpectError: regexp.MustCompile(`(?i)FILESYSTEM`),
			},
		},
	})
}

func testAccDatasetDataSourceConfig(path string) string {
	return fmt.Sprintf(`
resource "truenas_dataset" "test" {
  path        = %q
  comments    = "tf-acc datasource test"
  compression = "LZ4"
}

data "truenas_dataset" "test" {
  path = truenas_dataset.test.path
}
`, path)
}

func testAccDatasetDataSourceByPathOnlyConfig(path string) string {
	return fmt.Sprintf(`
data "truenas_dataset" "test" {
  path = %q
}
`, path)
}
