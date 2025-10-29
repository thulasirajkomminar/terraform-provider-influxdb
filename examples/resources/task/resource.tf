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

resource "influxdb_task" "test_cron" {
  org_id = data.influxdb_organization.iot.id
  flux   = <<-EOT
    option task = {
      name: "Test Cron Task",
      cron: "0 0 * * *"
    }

    from(bucket: "test-bucket")
        |> range(start: -1h)
        |> filter(fn: (r) => r._measurement == "cpu")
        |> mean()
        |> to(bucket: "output-bucket", org: "test-org")
  EOT
}

output "test" {
  value = influxdb_task.test_cron
}
