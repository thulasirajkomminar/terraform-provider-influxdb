package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &LabelResource{}
	_ resource.ResourceWithConfigure   = &LabelResource{}
	_ resource.ResourceWithImportState = &LabelResource{}
)

// NewLabelResource is a helper function to simplify the provider implementation.
func NewLabelResource() resource.Resource {
	return &LabelResource{}
}

// LabelResource defines the resource implementation.
type LabelResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *LabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_label"
}

// Schema defines the schema for the resource.
func (r *LabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a label.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The label ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "A label name.",
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "The organization ID.",
			},
			"properties": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "The key-value pairs to associate with this label.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *LabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan LabelModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createLabel := domain.LabelCreateRequest{
		Name:  plan.Name.ValueString(),
		OrgID: plan.OrgID.ValueString(),
	}

	// Convert properties map to domain format if provided
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		propertiesMap := make(map[string]string)
		for key, value := range plan.Properties.Elements() {
			if strVal, ok := value.(types.String); ok {
				propertiesMap[key] = strVal.ValueString()
			}
		}
		createLabel.Properties = &domain.LabelCreateRequest_Properties{
			AdditionalProperties: propertiesMap,
		}
	}

	createLabelResponse, err := r.client.LabelsAPI().CreateLabel(ctx, &createLabel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating label",
			"Could not create label, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Handle properties conversion using helper function
	propertiesMap, diags := convertLabelProperties(ctx, createLabelResponse.Properties)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringPointerValue(createLabelResponse.Id)
	plan.OrgID = types.StringPointerValue(createLabelResponse.OrgID)
	plan.Properties = propertiesMap

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *LabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state LabelModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed label value from InfluxDB
	label, err := r.client.LabelsAPI().FindLabelByID(ctx, state.Id.ValueString())
	if err != nil {
		// The label was deleted outside of Terraform: remove it from state
		// so Terraform plans a re-create.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError(
			"Error getting label",
			formatAPIError(err),
		)

		return
	}

	// Overwrite items with refreshed state
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

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *LabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan LabelModel
	var state LabelModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	propertiesMapUpdate := make(map[string]string)
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		// Properties are specified in the configuration
		for key, value := range plan.Properties.Elements() {
			if strVal, ok := value.(types.String); ok {
				propertiesMapUpdate[key] = strVal.ValueString()
			}
		}

		// Check for properties that exist in current state but not in plan
		// These should be removed by setting them to empty string
		if !state.Properties.IsNull() && !state.Properties.IsUnknown() {
			for key := range state.Properties.Elements() {
				if _, exists := propertiesMapUpdate[key]; !exists {
					// Property was removed from config, send empty string to remove it
					propertiesMapUpdate[key] = ""
				}
			}
		}
	} else {
		// Properties block is completely removed from configuration
		// Remove all existing properties by setting them to empty strings
		if !state.Properties.IsNull() && !state.Properties.IsUnknown() {
			for key := range state.Properties.Elements() {
				propertiesMapUpdate[key] = ""
			}
		}
	}

	updateLabel := domain.Label{
		Id:         plan.Id.ValueStringPointer(),
		Name:       plan.Name.ValueStringPointer(),
		OrgID:      plan.OrgID.ValueStringPointer(),
		Properties: &domain.Label_Properties{AdditionalProperties: propertiesMapUpdate},
	}

	// Update existing label
	apiResponse, err := r.client.LabelsAPI().UpdateLabel(ctx, &updateLabel)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating label",
			"Could not update label, unexpected error: "+formatAPIError(err),
		)
		return
	}

	// Handle properties conversion based on the configuration
	if plan.Properties.IsNull() {
		// Properties block was removed from configuration, so state should be null
		plan.Properties = types.MapNull(types.StringType)
	} else {
		// Properties are specified in configuration, use converted response
		propertiesMap, diags := convertLabelProperties(ctx, apiResponse.Properties)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		plan.Properties = propertiesMap
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringPointerValue(apiResponse.Id)
	plan.Name = types.StringPointerValue(apiResponse.Name)
	plan.OrgID = types.StringPointerValue(apiResponse.OrgID)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *LabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state LabelModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing label
	err := r.client.LabelsAPI().DeleteLabelWithID(ctx, state.Id.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting label",
			"Could not delete label, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *LabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *LabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
