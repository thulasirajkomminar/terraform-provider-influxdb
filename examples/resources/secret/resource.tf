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

resource "influxdb_secret" "datadog_api_key" {
  org_id = data.influxdb_organization.iot.id
  key    = "DATADOG_API_KEY"
  value  = var.datadog_api_key
}

variable "datadog_api_key" {
  type      = string
  sensitive = true
}
