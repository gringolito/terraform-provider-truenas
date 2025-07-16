// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccGroupDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - valid group
			{
				Config: testAccGroupDataSourceConfig("root"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("group"),
						knownvalue.StringExact("root"),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("gid"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func TestAccGroupDataSource_NonExistent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccGroupDataSourceConfig("nonexistent-group-12345"),
				ExpectError: regexp.MustCompile("group not found"),
			},
		},
	})
}

func TestAccGroupDataSource_SystemGroups(t *testing.T) {
	testCases := []struct {
		groupName string
		expectGid int
	}{
		{"wheel", 0},
		{"daemon", 1},
		{"bin", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.groupName, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: testAccGroupDataSourceConfig(tc.groupName),
						ConfigStateChecks: []statecheck.StateCheck{
							statecheck.ExpectKnownValue(
								"data.truenas_group.test",
								tfjsonpath.New("group"),
								knownvalue.StringExact(tc.groupName),
							),
							statecheck.ExpectKnownValue(
								"data.truenas_group.test",
								tfjsonpath.New("gid"),
								knownvalue.Int64Exact(int64(tc.expectGid)),
							),
						},
					},
				},
			})
		})
	}
}

func TestAccGroupDataSource_AllAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		// ExternalProviders: map[string]resource.ExternalProvider{
		// 	"truenas": {
		// 		Source: "hashicorp.com/gringolito/truenas",
		// 	},
		// },
		Steps: []resource.TestStep{
			{
				Config: testAccGroupDataSourceConfig("root"),
				ConfigStateChecks: []statecheck.StateCheck{
					// Verify all expected attributes are present
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("id"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("gid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("group"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("name"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("builtin"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("local"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("smb"),
						knownvalue.Bool(true),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("sid"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("userns_idmap"),
						knownvalue.NotNull(),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("sudo_commands"),
						knownvalue.ListExact([]knownvalue.Check{}),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("sudo_commands_nopasswd"),
						knownvalue.ListExact([]knownvalue.Check{}),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("roles"),
						knownvalue.ListExact([]knownvalue.Check{}),
					),
					statecheck.ExpectKnownValue(
						"data.truenas_group.test",
						tfjsonpath.New("users"),
						knownvalue.ListExact([]knownvalue.Check{}),
					),
				},
			},
		},
	})
}

func testAccGroupDataSourceConfig(groupName string) string {
	return fmt.Sprintf(`
terraform {
  required_providers {
    truenas = {
      source = "hashicorp.com/gringolito/truenas"
    }
  }
}

data "truenas_group" "test" {
  group = "%s"
}
`, groupName)
}
