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
	_ datasource.DataSource              = &OrganizationsDataSource{}
	_ datasource.DataSourceWithConfigure = &OrganizationsDataSource{}
)

// NewOrganizationsDataSource is a helper function to simplify the provider implementation.
func NewOrganizationsDataSource() datasource.DataSource {
	return &OrganizationsDataSource{}
}

// OrganizationsDataSource is the data source implementation.
type OrganizationsDataSource struct {
	client influxdb2.Client
}

// OrganizationsDataSourceModel describes the data source data model.
type OrganizationsDataSourceModel struct {
	Organizations []OrganizationModel `tfsdk:"organizations"`
}

// Metadata returns the data source type name.
func (d *OrganizationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organizations"
}

// Schema defines the schema for the data source.
func (d *OrganizationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Lists organizations. InfluxDB returns all organizations.",

		Attributes: map[string]schema.Attribute{
			"organizations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "An organization ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *OrganizationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *OrganizationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state OrganizationsDataSourceModel

	organizations, err := d.client.OrganizationsAPI().GetOrganizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list Organizations",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	for _, organization := range *organizations {
		var organizationState OrganizationModel
		populateOrganizationModel(&organizationState, &organization)

		state.Organizations = append(state.Organizations, organizationState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
