terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_labels" "all" {}

output "all_labels" {
  value = data.influxdb_labels.all
}
