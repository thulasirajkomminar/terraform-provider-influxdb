package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &OrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationDataSource{}
)

// NewOrganizationDataSource is a helper function to simplify the provider implementation.
func NewOrganizationDataSource() datasource.DataSource {
	return &OrganizationDataSource{}
}

// OrganizationDataSource is the data source implementation.
type OrganizationDataSource struct {
	client influxdb2.Client
}

// Metadata returns the data source type name.
func (d *OrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the data source.
func (d *OrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves an organization. Use this data source to retrieve information for a specific organization.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "An organization ID.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the organization.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Organization creation date in RFC3339 format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Last Organization update date in RFC3339 format.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *OrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state OrganizationModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := state.Name
	if orgName.IsNull() {
		resp.Diagnostics.AddError(
			"Name is empty",
			"Must set name",
		)

		return
	}

	organization, err := d.client.OrganizationsAPI().FindOrganizationByName(ctx, orgName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting organization",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	populateOrganizationModel(&state, organization)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
