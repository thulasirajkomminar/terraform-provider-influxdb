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

resource "influxdb_variable" "deployment" {
  org_id      = data.influxdb_organization.iot.id
  name        = "deployment"
  description = "The deployment environments"
  arguments = jsonencode({
    type   = "constant"
    values = ["production", "staging", "development"]
  })
}

resource "influxdb_variable" "buckets" {
  org_id = data.influxdb_organization.iot.id
  name   = "buckets"
  arguments = jsonencode({
    type     = "query"
    language = "flux"
    query    = "buckets() |> keep(columns: [\"name\"])"
  })
}
