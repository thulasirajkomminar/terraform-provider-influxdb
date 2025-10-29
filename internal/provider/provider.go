package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &InfluxDBProvider{}

// InfluxDBProvider defines the provider implementation.
type InfluxDBProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// InfluxDBProviderModel maps provider schema data to a Go type.
type InfluxDBProviderModel struct {
	URL      types.String `tfsdk:"url"`
	Token    types.String `tfsdk:"token"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

// Metadata returns the provider type name.
func (p *InfluxDBProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "influxdb"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *InfluxDBProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "InfluxDB provider to deploy and manage resources supported by InfluxDB.",

		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The InfluxDB server URL",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "An InfluxDB token string",
				Optional:    true,
				Sensitive:   true,
			},
			"username": schema.StringAttribute{
				Description: "The InfluxDB username",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The InfluxDB password",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a InfluxDB API client for data sources and resources.
func (p *InfluxDBProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config InfluxDBProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.URL.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Unknown InfluxDB URL",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB URL. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB_URL environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown InfluxDB Token",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB Token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB_TOKEN environment variable.",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown InfluxDB Username",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB Username. "+
				"Either target apply the source of the value first or set the value statically in the configuration.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown InfluxDB Password",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB Password. "+
				"Either target apply the source of the value first or set the value statically in the configuration.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	username := os.Getenv("INFLUXDB_USERNAME")
	password := os.Getenv("INFLUXDB_PASSWORD")

	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing InfluxDB URL",
			"The provider cannot create the InfluxDB client as there is a missing or empty value for the InfluxDB URL. "+
				"Set the url value in the configuration or use the INFLUXDB_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	// Validate authentication credentials - require either token OR username+password
	hasToken := token != ""
	hasUsername := username != ""
	hasPassword := password != ""
	hasCompleteUsernamePassword := hasUsername && hasPassword

	if !hasToken && !hasUsername && !hasPassword {
		// No authentication provided at all
		resp.Diagnostics.AddError(
			"Missing InfluxDB Authentication",
			"The provider cannot create the InfluxDB client as the authentication credentials are missing or empty.\n\n"+
				"Choose one of the following authentication methods:\n"+
				"• Token authentication: Set 'token' in configuration or use INFLUXDB_TOKEN environment variable.\n"+
				"• Password authentication: Set both 'username' and 'password' in configuration or use INFLUXDB_USERNAME & INFLUXDB_PASSWORD environment variable.",
		)
	} else if !hasToken && !hasCompleteUsernamePassword {
		// Partial username/password credentials provided
		if !hasUsername {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Incomplete InfluxDB Authentication",
				"Username is required when using username and password authentication. "+
					"Provide both username and password, or use token authentication instead.",
			)
		}
		if !hasPassword {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Incomplete InfluxDB Authentication",
				"Password is required when using username and password authentication. "+
					"Provide both username and password, or use token authentication instead.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "INFLUXDB_URL", url)
	ctx = tflog.SetField(ctx, "INFLUXDB_TOKEN", token)
	ctx = tflog.SetField(ctx, "INFLUXDB_USERNAME", username)
	ctx = tflog.SetField(ctx, "INFLUXDB_PASSWORD", password)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "INFLUXDB_TOKEN")
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "INFLUXDB_PASSWORD")

	tflog.Debug(ctx, "Creating InfluxDB client")

	// Create a new InfluxDB client using the configuration values
	// Token authentication takes priority over username/password
	var client influxdb2.Client

	if token != "" {
		// Use token authentication (priority)
		client = influxdb2.NewClient(url, token)
	} else {
		// Use username/password authentication (fallback)
		client = influxdb2.NewClientWithOptions(
			url,
			"",
			influxdb2.DefaultOptions(),
		)

		err := client.UsersAPI().SignIn(context.Background(), username, password)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create InfluxDB Client",
				"Failed to signin with username and password to InfluxDB.\n\n"+
					"InfluxDB Client Error: "+err.Error(),
			)
			return
		}
	}

	_, err := client.Ping(context.Background())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create InfluxDB Client",
			"An unexpected error occurred when creating the InfluxDB client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"InfluxDB Client Error: "+err.Error(),
		)
		return
	}

	// Make the InfluxDB client available during DataSource and Resource
	// type Configure methods.
	resp.DataSourceData = client
	resp.ResourceData = client

	tflog.Info(ctx, "Configured InfluxDB client", map[string]any{"success": true})
}

// Resources defines the resources implemented in the provider.
func (p *InfluxDBProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewAuthorizationResource,
		NewBucketResource,
		NewLabelResource,
		NewOrganizationResource,
		NewTaskResource,
		NewUserResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *InfluxDBProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAuthorizationDataSource,
		NewAuthorizationsDataSource,
		NewBucketDataSource,
		NewBucketsDataSource,
		NewLabelDataSource,
		NewLabelsDataSource,
		NewOrganizationDataSource,
		NewOrganizationsDataSource,
		NewTaskDataSource,
		NewTasksDataSource,
		NewUserDataSource,
		NewUsersDataSource,
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &InfluxDBProvider{
			version: version,
		}
	}
}
