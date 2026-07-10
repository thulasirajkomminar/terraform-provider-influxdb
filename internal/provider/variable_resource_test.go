package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccVariableResource(t *testing.T) {
	variableName := acctest.RandomWithPrefix("tf-variable-test")
	updatedName := acctest.RandomWithPrefix("tf-variable-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccVariableResourceConfig(variableName, "test variable", `{"type":"constant","values":["production","staging"]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("influxdb_variable.test", "id"),
					resource.TestCheckResourceAttr("influxdb_variable.test", "name", variableName),
					resource.TestCheckResourceAttr("influxdb_variable.test", "description", "test variable"),
					resource.TestCheckResourceAttr("influxdb_variable.test", "arguments", `{"type":"constant","values":["production","staging"]}`),
					resource.TestCheckResourceAttrSet("influxdb_variable.test", "created_at"),
					resource.TestCheckResourceAttrSet("influxdb_variable.test", "updated_at"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "influxdb_variable.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The API may re-encode the JSON arguments; they are compared
				// semantically at plan time, not byte-for-byte.
				ImportStateVerifyIgnore: []string{"arguments"},
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccVariableResourceConfig(updatedName, "updated variable", `{"type":"constant","values":["production","staging","development"]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_variable.test", "name", updatedName),
					resource.TestCheckResourceAttr("influxdb_variable.test", "description", "updated variable"),
					resource.TestCheckResourceAttr("influxdb_variable.test", "arguments", `{"type":"constant","values":["production","staging","development"]}`),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccVariableResourceInvalidArguments(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Invalid JSON must fail at plan time, before anything is created.
			{
				Config:      providerConfig + testAccVariableResourceConfig("tf-variable-invalid", "invalid", "not-json"),
				ExpectError: regexp.MustCompile("Invalid JSON String Value"),
			},
		},
	})
}

func testAccVariableResourceConfig(name, description, arguments string) string {
	return fmt.Sprintf(`
resource "influxdb_variable" "test" {
  org_id      = %[1]q
  name        = %[2]q
  description = %[3]q
  arguments   = %[4]q
}
`, os.Getenv("INFLUXDB_ORG_ID"), name, description, arguments)
}
