package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserDataSource(t *testing.T) {
	userName := acctest.RandomWithPrefix("tf-user-test")
	password := acctest.RandomWithPrefix("password")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - basic user without organization
			{
				Config: providerConfig + testAccUserDataSourceConfig(userName, password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.influxdb_user.test", "name", userName),
					resource.TestCheckResourceAttr("data.influxdb_user.test", "status", "active"),
					resource.TestCheckResourceAttrSet("data.influxdb_user.test", "id"),
					// Password should always be null in data source
					resource.TestCheckNoResourceAttr("data.influxdb_user.test", "password"),
					// Org fields should be null when user is not in an org
					resource.TestCheckNoResourceAttr("data.influxdb_user.test", "org_id"),
					resource.TestCheckNoResourceAttr("data.influxdb_user.test", "org_role"),
				),
			},
		},
	})
}

func TestAccUserDataSourceWithOrganization(t *testing.T) {
	userName := acctest.RandomWithPrefix("tf-user-test")
	password := acctest.RandomWithPrefix("password")
	orgName := acctest.RandomWithPrefix("tf-org-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing - user with organization membership
			{
				Config: providerConfig + testAccUserDataSourceWithOrgConfig(userName, password, orgName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.influxdb_user.test", "name", userName),
					resource.TestCheckResourceAttr("data.influxdb_user.test", "status", "active"),
					resource.TestCheckResourceAttrSet("data.influxdb_user.test", "id"),
					// Password should always be null in data source
					resource.TestCheckNoResourceAttr("data.influxdb_user.test", "password"),
					// Org fields should be set when user is in an org
					resource.TestCheckResourceAttrSet("data.influxdb_user.test", "org_id"),
					resource.TestCheckResourceAttr("data.influxdb_user.test", "org_role", "member"),
				),
			},
		},
	})
}

func TestAccUserDataSourceNonExistent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test error handling for non-existent user
			{
				Config:      providerConfig + testAccUserDataSourceNonExistentConfig,
				ExpectError: regexp.MustCompile("Error getting user"),
			},
		},
	})
}

func testAccUserDataSourceConfig(userName, password string) string {
	return fmt.Sprintf(`
resource "influxdb_user" "test" {
	name     = "%s"
	password = "%s"
	status   = "active"
}

data "influxdb_user" "test" {
	id = influxdb_user.test.id
	depends_on = [influxdb_user.test]
}
`, userName, password)
}

func testAccUserDataSourceWithOrgConfig(userName, password, orgName string) string {
	return fmt.Sprintf(`
resource "influxdb_organization" "test" {
	name        = "%s"
	description = "Test organization for user data source"
}

resource "influxdb_user" "test" {
	name     = "%s"
	password = "%s"
	status   = "active"
	org_id   = influxdb_organization.test.id
	org_role = "member"
}

data "influxdb_user" "test" {
	id = influxdb_user.test.id
	depends_on = [influxdb_user.test]
}
`, orgName, userName, password)
}

const testAccUserDataSourceNonExistentConfig = `
data "influxdb_user" "test" {
	id = "nonexistent-user-id-12345"
}
`
