package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestMain wires up sweeper support (`go test -sweep=<anything>`). This
// provider has no region concept; sweepers ignore the region argument and
// connect using the same TRUENAS_HOST/TRUENAS_API_KEY/... environment
// variables as the acceptance tests themselves.
func TestMain(m *testing.M) {
	resource.TestMain(m)
}
