package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &OrganizationResource{}
	_ resource.ResourceWithConfigure    = &OrganizationResource{}
	_ resource.ResourceWithImportState  = &OrganizationResource{}
	_ resource.ResourceWithUpgradeState = &OrganizationResource{}
)

// NewOrganizationResource is a helper function to simplify the provider implementation.
func NewOrganizationResource() resource.Resource {
	return &OrganizationResource{}
}

// OrganizationResource defines the resource implementation.
type OrganizationResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *OrganizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

// Schema defines the schema for the resource.
func (r *OrganizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Creates and manages new organization.",

		// Version 1 stores timestamps as RFC3339.
		Version: 1,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "An organization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
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

// UpgradeState migrates state written by prior provider versions: timestamps
// move from Go's time.Time.String() format to RFC3339.
func (r *OrganizationResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: timestampStateUpgrader("created_at", "updated_at"),
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrganizationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createOrganization := domain.Organization{
		Id:          plan.Id.ValueStringPointer(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	}

	apiResponse, err := r.client.OrganizationsAPI().CreateOrganization(ctx, &createOrganization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organization",
			"Could not create organization, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	populateOrganizationModel(&plan, apiResponse)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state OrganizationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed organization value from InfluxDB
	readOrganization, err := r.client.OrganizationsAPI().FindOrganizationByName(ctx, state.Name.ValueString())
	if err != nil {
		// The organization was deleted outside of Terraform: remove it from
		// state so Terraform plans a re-create.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError(
			"Error getting organization",
			formatAPIError(err),
		)

		return
	}

	// Overwrite items with refreshed state
	populateOrganizationModel(&state, readOrganization)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrganizationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	updateOrganization := domain.Organization{
		Id:          plan.Id.ValueStringPointer(),
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueStringPointer(),
	}

	// Update existing organization
	apiResponse, err := r.client.OrganizationsAPI().UpdateOrganization(ctx, &updateOrganization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization",
			"Could not update organization, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	populateOrganizationModel(&plan, apiResponse)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrganizationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing organization
	err := r.client.OrganizationsAPI().DeleteOrganizationWithID(ctx, state.Id.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting organization",
			"Could not delete organization, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *OrganizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
