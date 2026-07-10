package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// VariableModel maps InfluxDB variable schema data.
type VariableModel struct {
	Id          types.String         `tfsdk:"id"`
	OrgID       types.String         `tfsdk:"org_id"`
	Name        types.String         `tfsdk:"name"`
	Description types.String         `tfsdk:"description"`
	Arguments   jsontypes.Normalized `tfsdk:"arguments"`
	CreatedAt   timetypes.RFC3339    `tfsdk:"created_at"`
	UpdatedAt   timetypes.RFC3339    `tfsdk:"updated_at"`
}

// populateVariableModel fills the model from an API variable, guarding every
// optional field so a partial response can never panic and cleared values
// reset to null instead of lingering in state.
func populateVariableModel(model *VariableModel, variable *domain.Variable) diag.Diagnostics {
	var diags diag.Diagnostics

	model.Id = types.StringPointerValue(variable.Id)
	model.OrgID = types.StringValue(variable.OrgID)
	model.Name = types.StringValue(variable.Name)
	model.Description = types.StringPointerValue(variable.Description)
	model.CreatedAt = rfc3339PointerValue(variable.CreatedAt)
	model.UpdatedAt = rfc3339PointerValue(variable.UpdatedAt)

	if variable.Arguments == nil {
		model.Arguments = jsontypes.NewNormalizedNull()

		return diags
	}

	arguments, err := json.Marshal(variable.Arguments)
	if err != nil {
		diags.AddError(
			"Unable to Encode Variable Arguments",
			"An unexpected error occurred while encoding the variable arguments returned by the API: "+err.Error(),
		)

		return diags
	}

	model.Arguments = jsontypes.NewNormalizedValue(string(arguments))

	return diags
}

// buildVariableArguments decodes the configured JSON arguments into the API
// representation.
func buildVariableArguments(arguments jsontypes.Normalized) (domain.VariableProperties, diag.Diagnostics) {
	var diags diag.Diagnostics

	var properties domain.VariableProperties
	if err := json.Unmarshal([]byte(arguments.ValueString()), &properties); err != nil {
		diags.AddError(
			"Invalid Variable Arguments",
			"The arguments attribute must contain valid JSON: "+err.Error(),
		)

		return nil, diags
	}

	return properties, diags
}
