# terraform-provider-influxdb
Terraform provider to manage InfluxDB.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Add the below code to your configuration.

```terraform
terraform {
  required_providers {
    influxdb = {
      source = "thulasirajkomminar/influxdb"
    }
  }
}
```

### Initialize the provider

#### Token-based authentication

```terraform
provider "influxdb" {
  url   = "http://localhost:8086"
  token = "influxdb-token"
}
```

#### Username and password authentication

```terraform
provider "influxdb" {
  url      = "http://localhost:8086"
  username = "influxdb-user"
  password = "influxdb-password"
}
```

## Supported InfluxDB flavours

### v3

* [InfluxDB Cloud Serverless](https://www.influxdata.com/products/influxdb-cloud/serverless/)

### v2

* [InfluxDB Cloud TSM](https://docs.influxdata.com/influxdb/cloud/)
* [InfluxDB OSS](https://docs.influxdata.com/influxdb/v2/)
  
## Available functionalities

### Data Sources

* `influxdb_authorization`
* `influxdb_authorizations`
* `influxdb_bucket`
* `influxdb_buckets`
* `influxdb_organization`
* `influxdb_organizations`

### Resources

* `influxdb_authorization`
* `influxdb_bucket`
* `influxdb_organization`

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make docs`.

## Running Tests

Set the below environment variables to run the tests:

1. `INFLUXDB_URL`
2. `INFLUXDB_TOKEN` (for token-based authentication)
3. `INFLUXDB_USERNAME` and `INFLUXDB_PASSWORD` (for username/password authentication)
4. `INFLUXDB_ORG` (the organization to use for the tests)

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
