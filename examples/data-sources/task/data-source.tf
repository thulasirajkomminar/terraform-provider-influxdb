terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_task" "test" {
  id = "0f1da6d9c3471000"
}

output "test_task" {
  value = data.influxdb_task.test
}
