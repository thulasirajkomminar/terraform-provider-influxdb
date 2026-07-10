package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariablesDataSource(t *testing.T) {
	orgName := acctest.RandomWithPrefix("tf-org-test")
	variableName := acctest.RandomWithPrefix("tf-variable-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// A dedicated organization guarantees a deterministic listing.
			{
				Config: providerConfig + testAccVariablesDataSourceConfig(orgName, variableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.influxdb_variables.test", "variables.#", "1"),
					resource.TestCheckResourceAttr("data.influxdb_variables.test", "variables.0.name", variableName),
					resource.TestCheckResourceAttrSet("data.influxdb_variables.test", "variables.0.id"),
					resource.TestCheckResourceAttrSet("data.influxdb_variables.test", "variables.0.arguments"),
				),
			},
		},
	})
}

func testAccVariablesDataSourceConfig(orgName, variableName string) string {
	return fmt.Sprintf(`
resource "influxdb_organization" "test" {
  name        = %[1]q
  description = "Test organization for variables data source"
}

resource "influxdb_variable" "test" {
  org_id    = influxdb_organization.test.id
  name      = %[2]q
  arguments = jsonencode({
    type   = "constant"
    values = ["a", "b"]
  })
}

data "influxdb_variables" "test" {
  org_id     = influxdb_organization.test.id
  depends_on = [influxdb_variable.test]
}
`, orgName, variableName)
}
