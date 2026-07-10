package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &VariableDataSource{}
	_ datasource.DataSourceWithConfigure = &VariableDataSource{}
)

// NewVariableDataSource is a helper function to simplify the provider implementation.
func NewVariableDataSource() datasource.DataSource {
	return &VariableDataSource{}
}

// VariableDataSource is the data source implementation.
type VariableDataSource struct {
	client influxdb2.Client
}

// Metadata returns the data source type name.
func (d *VariableDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

// Schema defines the schema for the data source.
func (d *VariableDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves a dashboard variable.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The variable ID.",
			},
			"org_id": schema.StringAttribute{
				Computed:    true,
				Description: "The organization ID that owns the variable.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The variable name.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The variable description.",
			},
			"arguments": schema.StringAttribute{
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "The variable arguments as a JSON object.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Variable creation date in RFC3339 format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Last variable update date in RFC3339 format.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *VariableDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *VariableDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VariableModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variable, err := d.client.APIClient().GetVariablesID(ctx, &domain.GetVariablesIDAllParams{
		VariableID: state.Id.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting variable",
			formatAPIError(err),
		)

		return
	}

	if variable == nil {
		resp.Diagnostics.AddError(
			"Error getting variable",
			"The API returned an empty variable response.",
		)

		return
	}

	// Map response body to model
	resp.Diagnostics.Append(populateVariableModel(&state, variable)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
