package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAuthorizationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccAuthorizationResourceConfig("Access test bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_authorization.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("influxdb_authorization.test", "description", "Access test bucket"),
				),
			},
			// ImportState testing
			{
				ResourceName: "influxdb_authorization.test",
				ImportState:  true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccAuthorizationResourceConfig("RW access test bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_authorization.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("influxdb_authorization.test", "description", "RW access test bucket"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccAuthorizationResourcePermissionOrder verifies that permissions are
// set-typed: reordering the permission blocks in the configuration must not
// produce a diff.
func TestAccAuthorizationResourcePermissionOrder(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccAuthorizationResourceOrderedConfig("read", "write"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_authorization.test", "permissions.#", "2"),
				),
			},
			// The same permissions in reverse order must plan clean.
			{
				Config:   providerConfig + testAccAuthorizationResourceOrderedConfig("write", "read"),
				PlanOnly: true,
			},
		},
	})
}

func testAccAuthorizationResourceOrderedConfig(firstAction, secondAction string) string {
	return fmt.Sprintf(`
resource "influxdb_bucket" "test" {
	name = "test-permission-order"
	org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
  }

resource "influxdb_authorization" "test" {
	org_id      = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
	description = "permission order test"

	permissions = [{
	  action = %[1]q
	  resource = {
	  	org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
		id     = influxdb_bucket.test.id
		type   = "buckets"
	  }
	  },
	  {
		action = %[2]q
		resource = {
		  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
		  id     = influxdb_bucket.test.id
		  type   = "buckets"
		}
	}]
  }
`, firstAction, secondAction)
}

func testAccAuthorizationResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "influxdb_bucket" "test" {
	name = "test"
	org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
  }

resource "influxdb_authorization" "test" {
	org_id      = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
	description = %[1]q

	permissions = [{
	  action = "read"
	  resource = {
	  	org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
		id     = influxdb_bucket.test.id
		type   = "buckets"
	  }
	  },
	  {
		action = "write"
		resource = {
		  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
		  id     = influxdb_bucket.test.id
		  type   = "buckets"
		}
	}]
  }
`, description)
}
