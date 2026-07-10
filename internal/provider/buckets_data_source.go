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
	_ datasource.DataSource              = &BucketsDataSource{}
	_ datasource.DataSourceWithConfigure = &BucketsDataSource{}
)

// NewBucketsDataSource is a helper function to simplify the provider implementation.
func NewBucketsDataSource() datasource.DataSource {
	return &BucketsDataSource{}
}

// BucketsDataSource is the data source implementation.
type BucketsDataSource struct {
	client influxdb2.Client
}

// BucketsDataSourceModel describes the data source data model.
type BucketsDataSourceModel struct {
	Buckets []BucketModel `tfsdk:"buckets"`
}

// Metadata returns the data source type name.
func (d *BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buckets"
}

// Schema defines the schema for the data source.
func (d *BucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Lists buckets. InfluxDB retrieves buckets owned by the organization associated with the authorization (API token).",

		Attributes: map[string]schema.Attribute{
			"buckets": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "A Bucket ID.",
						},
						"org_id": schema.StringAttribute{
							Computed:    true,
							Description: "An organization ID.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The Bucket type.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "A description of the bucket.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "A Bucket name.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "Bucket creation date in RFC3339 format.",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "Last bucket update date in RFC3339 format.",
						},
						"retention_period": schema.Int64Attribute{
							Computed:    true,
							Description: "The duration in seconds for how long data will be kept in the database. `0` represents infinite retention.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *BucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state BucketsDataSourceModel

	buckets, err := d.client.BucketsAPI().GetBuckets(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list buckets",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	for _, bucket := range *buckets {
		var bucketState BucketModel
		populateBucketModel(&bucketState, &bucket)

		state.Buckets = append(state.Buckets, bucketState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
