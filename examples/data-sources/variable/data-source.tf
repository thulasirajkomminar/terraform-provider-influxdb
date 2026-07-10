terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_variable" "deployment" {
  id = "0dfa2b8e6d2f0001"
}

output "deployment_variable" {
  value = data.influxdb_variable.deployment
}
