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
	_ datasource.DataSource              = &BucketDataSource{}
	_ datasource.DataSourceWithConfigure = &BucketDataSource{}
)

// NewBucketDataSource is a helper function to simplify the provider implementation.
func NewBucketDataSource() datasource.DataSource {
	return &BucketDataSource{}
}

// BucketsDataSource is the data source implementation.
type BucketDataSource struct {
	client influxdb2.Client
}

// Metadata returns the data source type name.
func (d *BucketDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

// Schema defines the schema for the data source.
func (d *BucketDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves a bucket. Use this data source to retrieve information for a specific bucket.",

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
				Required:    true,
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
	}
}

// Configure adds the provider configured client to the data source.
func (d *BucketDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *BucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state BucketModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucketName := state.Name
	if bucketName.IsNull() {
		resp.Diagnostics.AddError(
			"Name is empty",
			"Must set name",
		)

		return
	}

	bucket, err := d.client.BucketsAPI().FindBucketByName(ctx, bucketName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting bucket",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	populateBucketModel(&state, bucket)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
