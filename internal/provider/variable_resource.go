package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
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
	_ resource.Resource                = &VariableResource{}
	_ resource.ResourceWithConfigure   = &VariableResource{}
	_ resource.ResourceWithImportState = &VariableResource{}
)

// NewVariableResource is a helper function to simplify the provider implementation.
func NewVariableResource() resource.Resource {
	return &VariableResource{}
}

// VariableResource defines the resource implementation.
type VariableResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *VariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_variable"
}

// Schema defines the schema for the resource.
func (r *VariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Creates and manages a dashboard variable.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The variable ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "The organization ID that owns the variable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The variable name.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The variable description.",
			},
			"arguments": schema.StringAttribute{
				Required:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "The variable arguments as a JSON object, e.g. `{\"type\": \"constant\", \"values\": [\"a\", \"b\"]}`. Must be valid JSON; this is validated at plan time.",
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

// Create creates the resource and sets the initial Terraform state.
func (r *VariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan VariableModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arguments, diags := buildVariableArguments(plan.Arguments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createVariable := domain.PostVariablesAllParams{
		Body: domain.PostVariablesJSONRequestBody{
			Name:        plan.Name.ValueString(),
			OrgID:       plan.OrgID.ValueString(),
			Description: plan.Description.ValueStringPointer(),
			Arguments:   arguments,
		},
	}

	apiResponse, err := r.client.APIClient().PostVariables(ctx, &createVariable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating variable",
			"Could not create variable, unexpected error: "+formatAPIError(err),
		)

		return
	}

	if apiResponse == nil {
		resp.Diagnostics.AddError(
			"Error creating variable",
			"The API returned an empty variable response.",
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	resp.Diagnostics.Append(populateVariableModel(&plan, apiResponse)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *VariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state VariableModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed variable value from InfluxDB
	variable, err := r.client.APIClient().GetVariablesID(ctx, &domain.GetVariablesIDAllParams{
		VariableID: state.Id.ValueString(),
	})
	if err != nil {
		// The variable was deleted outside of Terraform: remove it from
		// state so Terraform plans a re-create.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

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

	// Overwrite items with refreshed state
	resp.Diagnostics.Append(populateVariableModel(&state, variable)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *VariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan VariableModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	arguments, diags := buildVariableArguments(plan.Arguments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	updateVariable := domain.PatchVariablesIDAllParams{
		VariableID: plan.Id.ValueString(),
		Body: domain.PatchVariablesIDJSONRequestBody{
			Name:        plan.Name.ValueString(),
			OrgID:       plan.OrgID.ValueString(),
			Description: plan.Description.ValueStringPointer(),
			Arguments:   arguments,
		},
	}

	apiResponse, err := r.client.APIClient().PatchVariablesID(ctx, &updateVariable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating variable",
			"Could not update variable, unexpected error: "+formatAPIError(err),
		)

		return
	}

	if apiResponse == nil {
		resp.Diagnostics.AddError(
			"Error updating variable",
			"The API returned an empty variable response.",
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	resp.Diagnostics.Append(populateVariableModel(&plan, apiResponse)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *VariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state VariableModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing variable
	err := r.client.APIClient().DeleteVariablesID(ctx, &domain.DeleteVariablesIDAllParams{
		VariableID: state.Id.ValueString(),
	})
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting variable",
			"Could not delete variable, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *VariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *VariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
