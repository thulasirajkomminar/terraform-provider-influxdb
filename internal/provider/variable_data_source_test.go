package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableDataSource(t *testing.T) {
	variableName := acctest.RandomWithPrefix("tf-variable-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccVariableDataSourceConfig(variableName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.influxdb_variable.test", "id"),
					resource.TestCheckResourceAttr("data.influxdb_variable.test", "name", variableName),
					resource.TestCheckResourceAttr("data.influxdb_variable.test", "description", "data source test variable"),
					resource.TestCheckResourceAttr("data.influxdb_variable.test", "org_id", os.Getenv("INFLUXDB_ORG_ID")),
					resource.TestCheckResourceAttrSet("data.influxdb_variable.test", "arguments"),
					resource.TestCheckResourceAttrSet("data.influxdb_variable.test", "created_at"),
				),
			},
		},
	})
}

func testAccVariableDataSourceConfig(name string) string {
	return fmt.Sprintf(`
resource "influxdb_variable" "test" {
  org_id      = %[1]q
  name        = %[2]q
  description = "data source test variable"
  arguments   = jsonencode({
    type   = "constant"
    values = ["a", "b"]
  })
}

data "influxdb_variable" "test" {
  id = influxdb_variable.test.id
}
`, os.Getenv("INFLUXDB_ORG_ID"), name)
}
