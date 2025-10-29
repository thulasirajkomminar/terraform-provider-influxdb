terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_authorizations" "all" {}

output "all_authorizations" {
  value = data.influxdb_authorizations.all.authorizations[*].id
}
