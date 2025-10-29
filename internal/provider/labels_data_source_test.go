package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLabelsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create multiple labels and then read them with data source
			{
				Config: providerConfig + testAccLabelsDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check that we have labels (at least the ones we created)
					resource.TestCheckResourceAttrSet("data.influxdb_labels.test", "labels.#"),
					// Verify the created labels exist in the data source
					resource.TestCheckTypeSetElemNestedAttrs("data.influxdb_labels.test", "labels.*", map[string]string{
						"name": "test-labels-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.influxdb_labels.test", "labels.*", map[string]string{
						"name": "test-labels-2",
					}),
				),
			},
		},
	})
}

func TestAccLabelsDataSourceWithProperties(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create labels with properties and then read them with data source
			{
				Config: providerConfig + testAccLabelsDataSourceWithPropertiesConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Check that we have labels
					resource.TestCheckResourceAttrSet("data.influxdb_labels.test", "labels.#"),
					// Verify labels with properties exist
					resource.TestCheckTypeSetElemNestedAttrs("data.influxdb_labels.test", "labels.*", map[string]string{
						"name": "test-labels-props-1",
					}),
					resource.TestCheckTypeSetElemNestedAttrs("data.influxdb_labels.test", "labels.*", map[string]string{
						"name": "test-labels-props-2",
					}),
				),
			},
		},
	})
}

func testAccLabelsDataSourceConfig() string {
	return `
resource "influxdb_label" "test1" {
  name   = "test-labels-1"
  org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
}

resource "influxdb_label" "test2" {
  name   = "test-labels-2"
  org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
}

data "influxdb_labels" "test" {
  depends_on = [influxdb_label.test1, influxdb_label.test2]
}
`
}

func testAccLabelsDataSourceWithPropertiesConfig() string {
	return `
resource "influxdb_label" "test1" {
  name   = "test-labels-props-1"
  org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
  properties = {
	"environment" = "test"
	"team"        = "qa"
  }
}

resource "influxdb_label" "test2" {
  name   = "test-labels-props-2"
  org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
  properties = {
	"environment" = "production"
	"priority"    = "high"
  }
}

data "influxdb_labels" "test" {
  depends_on = [influxdb_label.test1, influxdb_label.test2]
}
`
}
