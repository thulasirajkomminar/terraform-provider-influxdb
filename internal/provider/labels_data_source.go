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
	_ datasource.DataSource              = &LabelsDataSource{}
	_ datasource.DataSourceWithConfigure = &LabelsDataSource{}
)

// NewLabelsDataSource is a helper function to simplify the provider implementation.
func NewLabelsDataSource() datasource.DataSource {
	return &LabelsDataSource{}
}

// LabelsDataSource is the data source implementation.
type LabelsDataSource struct {
	client influxdb2.Client
}

// LabelsDataSourceModel describes the data source data model.
type LabelsDataSourceModel struct {
	Labels []LabelModel `tfsdk:"labels"`
}

// Metadata returns the data source type name.
func (d *LabelsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_labels"
}

// Schema defines the schema for the data source.
func (d *LabelsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "List all labels.",

		Attributes: map[string]schema.Attribute{
			"labels": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *LabelsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *LabelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state LabelsDataSourceModel

	labels, err := d.client.LabelsAPI().GetLabels(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list labels",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	for _, label := range *labels {
		// Handle properties conversion using helper function
		propertiesMap, diags := convertLabelProperties(ctx, label.Properties)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		labelState := LabelModel{
			Id:         types.StringPointerValue(label.Id),
			Name:       types.StringPointerValue(label.Name),
			OrgID:      types.StringPointerValue(label.OrgID),
			Properties: propertiesMap,
		}

		state.Labels = append(state.Labels, labelState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
