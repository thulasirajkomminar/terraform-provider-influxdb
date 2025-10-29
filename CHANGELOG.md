# Changelog

All notable changes to this project will automatically be documented in this file.

The format is based on vKeep a Changelog(https://keepachangelog.com/en/1.0.0/),
and this project adheres to vSemantic Versioning(https://semver.org/spec/v2.0.0.html).

## v2.0.0 - 2025-10-29

### What's Changed

* chore: moved the repo from org to personal account.

> [!Important]
>
> The older versions of the provider are still available under the `komminarlabs/influxdb` namespace on the Terraform Registry. But the new versions (`v2.0.0` and above) will be available under the `thulasirajkomminar/influxdb` namespace. Please update your provider source accordingly in your Terraform configurations.

## v1.4.1 - 2025-07-14

### What's Changed

* fix: Remove validator for duplicate permissions in authorization reso… by @thulasirajkomminar in https://github.com/komminarlabs/terraform-provider-influxdb/pull/87

**Full Changelog**: https://github.com/komminarlabs/terraform-provider-influxdb/compare/v1.4.0...v1.4.1

## v1.4.0 - 2025-07-02

### What's Changed

* feat: Add InfluxDB task and label resources with data sources by @thulasirajkomminar in https://github.com/komminarlabs/terraform-provider-influxdb/pull/83
* feat: Add InfluxDB user resource with data sources by @thulasirajkomminar in https://github.com/komminarlabs/terraform-provider-influxdb/pull/84

**Full Changelog**: https://github.com/komminarlabs/terraform-provider-influxdb/compare/v1.3.0...v1.4.0

## v1.3.0 - 2025-06-19

### What's Changed

* Files Sync From komminarlabs/github-workflows by @komminarlabs-bot in https://github.com/komminarlabs/terraform-provider-influxdb/pull/76
* chore(deps): bump golang.org/x/net from 0.34.0 to 0.36.0 in the go_modules group by @dependabot in https://github.com/komminarlabs/terraform-provider-influxdb/pull/74
* feat: add dual authentication support with improved validation by @arpad42 in https://github.com/komminarlabs/terraform-provider-influxdb/pull/79
* chore: update Go dependencies by @thulasirajkomminar in https://github.com/komminarlabs/terraform-provider-influxdb/pull/80
* feat: added changelog action by @thulasirajkomminar in https://github.com/komminarlabs/terraform-provider-influxdb/pull/81

### New Contributors

* @arpad42 made their first contribution in https://github.com/komminarlabs/terraform-provider-influxdb/pull/79

**Full Changelog**: https://github.com/komminarlabs/terraform-provider-influxdb/compare/v1.2.0...v1.3.0

## v1.1.2 - 2024-09-12

## Updated:

* Updated docs to include supported influxdb flavours.

## v1.1.1 - 2024-08-23

## Fixed:

* fixed overwriting the token value during read in `influxdb_authorization` resource.

## v1.1.0 - 2024-04-16

## Updated:

* Updated `influxdb_authorization` resource and made the `id` & `org_id` as optional in `permissions.resource` inline with Influx api.
* Updated `influxdb_authorization` resource and made the `name` as read-only in `permissions.resource`. This is due to how Influx api returns the response. This will be modified in the future versions.

## v1.0.1 - 2024-03-05

## Updated:

* Updated provider docs
* Added Issue Templates
* Bump github.com/hashicorp/terraform-plugin-go from `0.21.0` to `0.22.0`
* Bump github.com/hashicorp/terraform-plugin-framework from `1.5.0` to `1.6.0`
* Upgrade golang
* Renamed module in `go.mod`

## v1.0.0 - 2024-02-27

## Added:

* Added an optional attribute `name` to `influxdb_authorization` resource.
* Acceptance tests for all data sources and resources.

## Updated:

* `retention_days` is renamed to `retention_period` in `influxdb_bucket` resource.
* Made some document changes.

## v0.1.0 - 2024-02-14

### Added:

* **New Data Source:** `influxdb_authorization`
* **New Data Source:** `influxdb_authorizations`
* **New Data Source:** `influxdb_bucket`
* **New Data Source:** `influxdb_buckets`
* **New Data Source:** `influxdb_organization`
* **New Data Source:** `influxdb_organizations`
* **New Resource:** `influxdb_authorization`
* **New Resource:** `influxdb_bucket`
* **New Resource:** `influxdb_organization`
