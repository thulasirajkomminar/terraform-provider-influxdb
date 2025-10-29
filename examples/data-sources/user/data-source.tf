terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}

provider "influxdb" {}

data "influxdb_user" "test-user" {
  id = "0f1ea53e8b3f900"
}

output "test-user-name" {
  value = data.influxdb_user.test-user.name
}

output "test-user-role" {
  value = data.influxdb_user.test-user.org_role
}
