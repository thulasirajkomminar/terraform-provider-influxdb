terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_buckets" "all" {}

output "all_buckets" {
  value = data.influxdb_buckets.all
}
