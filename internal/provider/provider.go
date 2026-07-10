package provider

import (
	"context"
	"encoding/base64"
	"os"
	"regexp"
	"strings"

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
				Description: "The InfluxDB server URL, e.g. `http://localhost:8086`. Trailing slashes are ignored. Can also be set with the `INFLUXDB_URL` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "An InfluxDB API token string. Can also be set with the `INFLUXDB_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"username": schema.StringAttribute{
				Description: "The InfluxDB username, used together with `password` when no token is configured. Can also be set with the `INFLUXDB_USERNAME` environment variable.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The InfluxDB password, used together with `username` when no token is configured. Can also be set with the `INFLUXDB_PASSWORD` environment variable.",
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
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB_USERNAME environment variable.",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown InfluxDB Password",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB Password. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB_PASSWORD environment variable.",
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

	// The InfluxDB client appends "api/v2/" to the URL, so normalize away
	// trailing slashes to avoid doubled separators.
	url = strings.TrimRight(url, "/")

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
				"• Password authentication: Set both 'username' and 'password' in configuration or use INFLUXDB_USERNAME & INFLUXDB_PASSWORD environment variables.",
		)
	} else if !hasToken && !hasCompleteUsernamePassword {
		// Partial username/password credentials provided
		if !hasUsername {
			resp.Diagnostics.AddAttributeError(
				path.Root("username"),
				"Missing InfluxDB Username",
				"Username is required when using username and password authentication. "+
					"Set 'username' in the configuration or use the INFLUXDB_USERNAME environment variable, "+
					"or use token authentication instead.",
			)
		}
		if !hasPassword {
			resp.Diagnostics.AddAttributeError(
				path.Root("password"),
				"Missing InfluxDB Password",
				"Password is required when using username and password authentication. "+
					"Set 'password' in the configuration or use the INFLUXDB_PASSWORD environment variable, "+
					"or use token authentication instead.",
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Build the masking rules for HTTP debug logging. Credentials must never
	// reach the logs, even when they appear in headers set above the logging
	// transport (Basic sign-in, session cookies) or in response bodies.
	var maskRegexes []*regexp.Regexp

	if hasToken {
		maskRegexes = append(maskRegexes, regexp.MustCompile(regexp.QuoteMeta(token)))
	}
	if hasPassword {
		maskRegexes = append(maskRegexes, regexp.MustCompile(regexp.QuoteMeta(password)))
	}
	if hasCompleteUsernamePassword {
		basicCredentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		maskRegexes = append(maskRegexes, regexp.MustCompile(regexp.QuoteMeta(basicCredentials)))
	}
	// Session cookies issued by /api/v2/signin.
	maskRegexes = append(maskRegexes, regexp.MustCompile(`influxdb-oss-session=[^;,\s]+`))

	// The token is injected by the authorization round tripper below the
	// logging transport, so the client itself is created without a token and
	// the Authorization header never reaches the debug logs.
	authorization := ""
	if hasToken {
		authorization = "Token " + token
	}

	options := influxdb2.DefaultOptions().SetHTTPClient(newHTTPClient(authorization, maskRegexes))

	tflog.Debug(ctx, "Creating InfluxDB client", map[string]any{"url": url})

	// Create a new InfluxDB client using the configuration values
	// Token authentication takes priority over username/password
	client := influxdb2.NewClientWithOptions(url, "", options)

	if !hasToken {
		// Use username/password authentication (fallback)
		err := client.UsersAPI().SignIn(ctx, username, password)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create InfluxDB Client",
				"Failed to signin with username and password to InfluxDB.\n\n"+
					"InfluxDB Client Error: "+formatAPIError(err),
			)
			return
		}
	}

	_, err := client.Ping(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create InfluxDB Client",
			"An unexpected error occurred when creating the InfluxDB client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"InfluxDB Client Error: "+formatAPIError(err),
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
		NewSecretResource,
		NewTaskResource,
		NewUserResource,
		NewVariableResource,
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
		NewVariableDataSource,
		NewVariablesDataSource,
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
