terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_organization" "iot" {
  name = "IoT"
}

resource "influxdb_label" "test" {
  name   = "test-label"
  org_id = data.influxdb_organization.iot.id
  properties = {
    description = "This is a test label"
  }
}

output "test" {
  value = influxdb_label.test
}
