package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &LabelDataSource{}
	_ datasource.DataSourceWithConfigure = &LabelDataSource{}
)

// NewLabelDataSource is a helper function to simplify the provider implementation.
func NewLabelDataSource() datasource.DataSource {
	return &LabelDataSource{}
}

// LabelDataSource is the data source implementation.
type LabelDataSource struct {
	client influxdb2.Client
}

// Metadata returns the data source type name.
func (d *LabelDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label"
}

// Schema defines the schema for the data source.
func (d *LabelDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves a label with label ID.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The label ID.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The label name.",
			},
			"org_id": schema.StringAttribute{
				Computed:    true,
				Description: "The organization ID.",
			},
			"properties": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The key-value pairs associated with this label.",
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *LabelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *LabelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state LabelModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	labelID := state.Id
	if labelID.IsNull() {
		resp.Diagnostics.AddError(
			"Id is empty",
			"Must set Id",
		)

		return
	}

	label, err := d.client.LabelsAPI().FindLabelByID(ctx, labelID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting label",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	// Handle properties conversion using helper function
	propertiesMap, diags := convertLabelProperties(ctx, label.Properties)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	state = LabelModel{
		Id:         types.StringPointerValue(label.Id),
		Name:       types.StringPointerValue(label.Name),
		OrgID:      types.StringPointerValue(label.OrgID),
		Properties: propertiesMap,
	}

	// Set state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
