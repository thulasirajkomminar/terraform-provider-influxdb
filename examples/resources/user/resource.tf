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

resource "influxdb_user" "test" {
  name     = "test-user"
  password = "test-password"
  org_id   = data.influxdb_organization.iot.id
  org_role = "owner"
}

output "test_user_id" {
  value = influxdb_user.test.id
}
