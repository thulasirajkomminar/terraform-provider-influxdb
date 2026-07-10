package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTaskResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test creating a task with cron schedule
			{
				Config: providerConfig + testAccTaskResourceConfigCron(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("influxdb_task.test_cron", "id"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_cron", "name"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_cron", "flux"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_cron", "org_id"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_cron", "created_at"),
				),
			},
			// Test creating a task with every schedule
			{
				Config: providerConfig + testAccTaskResourceConfigEvery(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("influxdb_task.test_every", "id"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_every", "name"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_every", "flux"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_every", "org_id"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_every", "created_at"),
				),
			},
			// ImportState testing
			{
				ResourceName: "influxdb_task.test_every",
				ImportState:  true,
			},
		},
	})
}

func TestAccTaskResourceValidation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test that tasks can be created with only Flux script (no validation errors)
			{
				Config: providerConfig + testAccTaskResourceConfigBasic(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("influxdb_task.test_basic", "id"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_basic", "name"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_basic", "flux"),
					resource.TestCheckResourceAttrSet("influxdb_task.test_basic", "org_id"),
				),
			},
		},
	})
}

func testAccTaskResourceConfigCron() string {
	return `
# First get the organization to use its ID
data "influxdb_organizations" "all" {}

resource "influxdb_task" "test_cron" {
  org_id = length(data.influxdb_organizations.all.organizations) > 0 ? data.influxdb_organizations.all.organizations[0].id : "dummy-org-id"
  flux   = <<-EOT
    option task = {
      name: "Test Cron Task",
      cron: "0 0 * * *",
    }
    
    from(bucket: "test-bucket")
      |> range(start: -1h)
      |> filter(fn: (r) => r._measurement == "cpu")
      |> mean()
      |> to(bucket: "output-bucket", org: "test-org")
  EOT
}
`
}

func testAccTaskResourceConfigEvery() string {
	return `
# First get the organization to use its ID
data "influxdb_organizations" "all" {}

resource "influxdb_task" "test_every" {
  org_id = length(data.influxdb_organizations.all.organizations) > 0 ? data.influxdb_organizations.all.organizations[0].id : "dummy-org-id"
  flux   = <<-EOT
    option task = {
      name: "Test Every Task",
      every: 1h,
    }
    
    from(bucket: "test-bucket")
      |> range(start: -1h)
      |> filter(fn: (r) => r._measurement == "memory")
      |> mean()
      |> to(bucket: "output-bucket", org: "test-org")
  EOT
}
`
}

func testAccTaskResourceConfigBasic() string {
	return `
# First get the organization to use its ID
data "influxdb_organizations" "all" {}

resource "influxdb_task" "test_basic" {
  org_id = length(data.influxdb_organizations.all.organizations) > 0 ? data.influxdb_organizations.all.organizations[0].id : "dummy-org-id"
  flux   = <<-EOT
    option task = {
      name: "Basic Test Task",
      every: 5m,
    }
    
    from(bucket: "test-bucket")
      |> range(start: -5m)
      |> filter(fn: (r) => r._measurement == "test")
      |> mean()
      |> to(bucket: "output-bucket", org: "test-org")
  EOT
}
`
}
