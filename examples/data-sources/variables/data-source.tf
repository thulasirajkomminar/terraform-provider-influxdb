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

data "influxdb_variables" "all" {
  org_id = data.influxdb_organization.iot.id
}

output "variables" {
  value = data.influxdb_variables.all.variables
}
