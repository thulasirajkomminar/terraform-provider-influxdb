package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &VariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &VariablesDataSource{}
)

// NewVariablesDataSource is a helper function to simplify the provider implementation.
func NewVariablesDataSource() datasource.DataSource {
	return &VariablesDataSource{}
}

// VariablesDataSource is the data source implementation.
type VariablesDataSource struct {
	client influxdb2.Client
}

// VariablesDataSourceModel describes the data source data model.
type VariablesDataSourceModel struct {
	OrgID     types.String    `tfsdk:"org_id"`
	Variables []VariableModel `tfsdk:"variables"`
}

// Metadata returns the data source type name.
func (d *VariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variables"
}

// Schema defines the schema for the data source.
func (d *VariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Lists dashboard variables.",

		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Optional:    true,
				Description: "The organization ID to filter variables by.",
			},
			"variables": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *VariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *VariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state VariablesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := domain.GetVariablesParams{}
	if !state.OrgID.IsNull() {
		params.OrgID = state.OrgID.ValueStringPointer()
	}

	variables, err := d.client.APIClient().GetVariables(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list variables",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	if variables != nil && variables.Variables != nil {
		for _, variable := range *variables.Variables {
			var variableState VariableModel
			resp.Diagnostics.Append(populateVariableModel(&variableState, &variable)...)
			if resp.Diagnostics.HasError() {
				return
			}

			state.Variables = append(state.Variables, variableState)
		}
	}

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
