terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_organizations" "all" {}

output "all_organizations" {
  value = data.influxdb_organizations.all
}
