package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSecretResource(t *testing.T) {
	secretKey := acctest.RandomWithPrefix("TF_SECRET_TEST")
	replacedKey := acctest.RandomWithPrefix("TF_SECRET_TEST")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccSecretResourceConfig(secretKey, "initial-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_secret.test", "key", secretKey),
					resource.TestCheckResourceAttr("influxdb_secret.test", "value", "initial-value"),
					resource.TestCheckResourceAttr("influxdb_secret.test", "org_id", os.Getenv("INFLUXDB_ORG_ID")),
				),
			},
			// Update the value in place (same key)
			{
				Config: providerConfig + testAccSecretResourceConfig(secretKey, "updated-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_secret.test", "key", secretKey),
					resource.TestCheckResourceAttr("influxdb_secret.test", "value", "updated-value"),
				),
			},
			// Changing the key forces a replacement
			{
				Config: providerConfig + testAccSecretResourceConfig(replacedKey, "updated-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_secret.test", "key", replacedKey),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccSecretResourceConfig(key, value string) string {
	return fmt.Sprintf(`
resource "influxdb_secret" "test" {
  org_id = %[1]q
  key    = %[2]q
  value  = %[3]q
}
`, os.Getenv("INFLUXDB_ORG_ID"), key, value)
}
