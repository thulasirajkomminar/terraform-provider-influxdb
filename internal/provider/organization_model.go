package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// OrganizationModel maps InfluxDB organization schema data.
type OrganizationModel struct {
	Id          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	CreatedAt   timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt   timetypes.RFC3339 `tfsdk:"updated_at"`
}

// populateOrganizationModel fills the model from an API organization,
// guarding every optional field so a partial response can never panic and
// cleared values reset to null instead of lingering in state.
func populateOrganizationModel(model *OrganizationModel, organization *domain.Organization) {
	model.Id = types.StringPointerValue(organization.Id)
	model.Name = types.StringValue(organization.Name)
	model.Description = types.StringPointerValue(organization.Description)
	model.CreatedAt = rfc3339PointerValue(organization.CreatedAt)
	model.UpdatedAt = rfc3339PointerValue(organization.UpdatedAt)
}
