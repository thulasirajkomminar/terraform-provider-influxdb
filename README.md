# terraform-provider-influxdb
Terraform provider to manage InfluxDB.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26 (to build the provider plugin)

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

#### Environment variables

Every provider attribute can also be set via an environment variable: `INFLUXDB_URL` (the server URL, trailing slashes are ignored), `INFLUXDB_TOKEN`, `INFLUXDB_USERNAME`, and `INFLUXDB_PASSWORD`. Values set in the provider configuration take precedence.

```shell
export INFLUXDB_URL="http://localhost:8086"
export INFLUXDB_TOKEN="influxdb-token"
```

```terraform
provider "influxdb" {}
```

## Supported InfluxDB flavours

### v3

* [InfluxDB Cloud Serverless](https://www.influxdata.com/products/influxdb-cloud/serverless/)

### v2

* [InfluxDB Cloud TSM](https://docs.influxdata.com/influxdb/cloud/)
* [InfluxDB OSS](https://docs.influxdata.com/influxdb/v2/)
  
## Available functionalities

### Data Sources

- `influxdb_authorization`
- `influxdb_authorizations`
- `influxdb_bucket`
- `influxdb_buckets`
- `influxdb_label`
- `influxdb_labels`
- `influxdb_organization`
- `influxdb_organizations`
- `influxdb_task`
- `influxdb_tasks`
- `influxdb_user`
- `influxdb_users`
- `influxdb_variable`
- `influxdb_variables`

### Resources

- `influxdb_authorization`
- `influxdb_bucket`
- `influxdb_label`
- `influxdb_organization`
- `influxdb_secret`
- `influxdb_task`
- `influxdb_user`
- `influxdb_variable`

## Debugging

Run Terraform with [`TF_LOG=DEBUG`](https://developer.hashicorp.com/terraform/internals/debugging) (or `TF_LOG_PROVIDER=DEBUG`) to log every HTTP request and response the provider exchanges with the InfluxDB API. Credentials never appear in these logs: the `Authorization` header is added below the logging layer, and configured tokens and passwords are masked (`***`) wherever they would otherwise appear, including sign-in headers, session cookies, and response bodies.

```shell
TF_LOG=DEBUG terraform plan
```

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make docs`.

## Running Tests

Set the below environment variables to run the tests:

1. `INFLUXDB_URL`
2. `INFLUXDB_TOKEN` (for token-based authentication)
3. `INFLUXDB_USERNAME` and `INFLUXDB_PASSWORD` (for username/password authentication)
4. `INFLUXDB_ORG_ID` (the organization to use for the tests)

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
