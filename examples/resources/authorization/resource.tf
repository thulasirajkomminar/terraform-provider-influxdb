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

data "influxdb_bucket" "signals" {
  name = "signals"
}

resource "influxdb_authorization" "signals" {
  org_id      = data.influxdb_organization.iot.id
  description = "Access signals bucket"

  permissions = [{
    action = "read"
    resource = {
      id     = data.influxdb_bucket.signals.id
      org_id = data.influxdb_organization.iot.id
      type   = "buckets"
    }
    },
    {
      action = "write"
      resource = {
        id     = data.influxdb_bucket.signals.id
        org_id = data.influxdb_organization.iot.id
        type   = "buckets"
      }
  }]
}

output "sample_authorization" {
  value = influxdb_authorization.signals.id
}
