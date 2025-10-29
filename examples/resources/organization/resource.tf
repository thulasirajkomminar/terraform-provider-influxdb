terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

resource "influxdb_organization" "iot" {
  name        = "IoT"
  description = "This is a sample organization"
}

output "sample_organization" {
  value = influxdb_organization.iot
}
