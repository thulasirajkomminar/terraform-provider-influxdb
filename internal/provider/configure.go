package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// configureResourceClient extracts the InfluxDB client from the provider data
// shared with resources. It returns nil (without diagnostics) when the
// provider has not been configured yet.
func configureResourceClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) influxdb2.Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(influxdb2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected influxdb2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil
	}

	return client
}

// configureDataSourceClient extracts the InfluxDB client from the provider
// data shared with data sources. It returns nil (without diagnostics) when
// the provider has not been configured yet.
func configureDataSourceClient(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) influxdb2.Client {
	if req.ProviderData == nil {
		return nil
	}

	client, ok := req.ProviderData.(influxdb2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return nil
	}

	return client
}
