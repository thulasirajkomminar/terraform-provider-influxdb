package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &BucketResource{}
	_ resource.ResourceWithConfigure    = &BucketResource{}
	_ resource.ResourceWithImportState  = &BucketResource{}
	_ resource.ResourceWithUpgradeState = &BucketResource{}
)

// NewBucketResource is a helper function to simplify the provider implementation.
func NewBucketResource() resource.Resource {
	return &BucketResource{}
}

// BucketResource defines the resource implementation.
type BucketResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *BucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

// Schema defines the schema for the resource.
func (r *BucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a bucket.",

		// Version 1 stores timestamps as RFC3339.
		Version: 1,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "A Bucket ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "An organization ID.",
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The Bucket type. Valid values are `user` or `system`.",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"user", "system"}...),
				},
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
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
			"retention_period": schema.Int64Attribute{ // buckets cannot have more than one retention rule at this time
				Computed:    true,
				Optional:    true,
				Default:     int64default.StaticInt64(2592000),
				Description: "The duration in seconds for how long data will be kept in the database. The default duration is `2592000` (30 days). `0` represents infinite retention.",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
	}
}

// UpgradeState migrates state written by prior provider versions: timestamps
// move from Go's time.Time.String() format to RFC3339.
func (r *BucketResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: timestampStateUpgrader("created_at", "updated_at"),
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *BucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BucketModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createBucket := domain.Bucket{
		OrgID:       plan.OrgID.ValueStringPointer(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		RetentionRules: []domain.RetentionRule{{
			EverySeconds: plan.RetentionPeriod.ValueInt64(),
		}},
	}

	apiResponse, err := r.client.BucketsAPI().CreateBucket(ctx, &createBucket)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating bucket",
			"Could not create bucket, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	populateBucketModel(&plan, apiResponse)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *BucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state BucketModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed bucket value from InfluxDB
	readBucket, err := r.client.BucketsAPI().FindBucketByID(ctx, state.Id.ValueString())
	if err != nil {
		// The bucket was deleted outside of Terraform: remove it from state
		// so Terraform plans a re-create.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError(
			"Error getting bucket",
			formatAPIError(err),
		)

		return
	}

	// Overwrite items with refreshed state
	populateBucketModel(&state, readBucket)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *BucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BucketModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	updateBucket := domain.Bucket{
		OrgID:       plan.OrgID.ValueStringPointer(),
		Id:          plan.Id.ValueStringPointer(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
		RetentionRules: []domain.RetentionRule{{
			EverySeconds: plan.RetentionPeriod.ValueInt64(),
		}},
	}

	// Update existing bucket
	apiResponse, err := r.client.BucketsAPI().UpdateBucket(ctx, &updateBucket)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating bucket",
			"Could not update bucket, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	populateBucketModel(&plan, apiResponse)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *BucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BucketModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing bucket
	err := r.client.BucketsAPI().DeleteBucketWithID(ctx, state.Id.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting bucket",
			"Could not delete bucket, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *BucketResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *BucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
