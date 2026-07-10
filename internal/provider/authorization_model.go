package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// AuthorizationModel maps InfluxDB authorization schema data.
type AuthorizationModel struct {
	Id          types.String                   `tfsdk:"id"`
	Token       types.String                   `tfsdk:"token"`
	Status      types.String                   `tfsdk:"status"`
	Description types.String                   `tfsdk:"description"`
	OrgID       types.String                   `tfsdk:"org_id"`
	Org         types.String                   `tfsdk:"org"`
	UserID      types.String                   `tfsdk:"user_id"`
	User        types.String                   `tfsdk:"user"`
	CreatedAt   timetypes.RFC3339              `tfsdk:"created_at"`
	UpdatedAt   timetypes.RFC3339              `tfsdk:"updated_at"`
	Permissions []AuthorizationPermissionModel `tfsdk:"permissions"`
}

// AuthorizationPermissionModel maps InfluxDB authorization permission schema data.
type AuthorizationPermissionModel struct {
	Action   types.String                         `tfsdk:"action"`
	Resource AuthorizationPermissionResourceModel `tfsdk:"resource"`
}

// AuthorizationPermissionResourceModel maps InfluxDB authorization permission resource schema data.
type AuthorizationPermissionResourceModel struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Org   types.String `tfsdk:"org"`
	OrgID types.String `tfsdk:"org_id"`
	Type  types.String `tfsdk:"type"`
}

// getPermissions converts API permissions to their model representation.
func getPermissions(permissions []domain.Permission) []AuthorizationPermissionModel {
	permissionsState := []AuthorizationPermissionModel{}
	for _, permission := range permissions {
		permissionState := AuthorizationPermissionModel{
			Action: types.StringValue(string(permission.Action)),
			Resource: AuthorizationPermissionResourceModel{
				Id:    types.StringPointerValue(permission.Resource.Id),
				Name:  types.StringPointerValue(permission.Resource.Name),
				Type:  types.StringValue(string(permission.Resource.Type)),
				OrgID: types.StringPointerValue(permission.Resource.OrgID),
				Org:   types.StringPointerValue(permission.Resource.Org),
			},
		}

		permissionsState = append(permissionsState, permissionState)
	}

	return permissionsState
}

// buildPermissions converts model permissions to the API request representation.
func buildPermissions(permissions []AuthorizationPermissionModel) []domain.Permission {
	var domainPermissions []domain.Permission
	for _, permissionData := range permissions {
		domainPermissions = append(domainPermissions, domain.Permission{
			Action: domain.PermissionAction(permissionData.Action.ValueString()),
			Resource: domain.Resource{
				Id:    permissionData.Resource.Id.ValueStringPointer(),
				Name:  permissionData.Resource.Name.ValueStringPointer(),
				Type:  domain.ResourceType(permissionData.Resource.Type.ValueString()),
				Org:   permissionData.Resource.Org.ValueStringPointer(),
				OrgID: permissionData.Resource.OrgID.ValueStringPointer(),
			},
		})
	}

	return domainPermissions
}

// populateAuthorizationModel fills the shared model fields from an API
// authorization, guarding every optional field so a partial response can
// never panic and cleared values reset to null instead of lingering in
// state. Token and status are intentionally left to the callers: the token
// is only returned on create, and the status must keep the practitioner
// configured value during create and update.
func populateAuthorizationModel(model *AuthorizationModel, authorization *domain.Authorization) {
	model.Id = types.StringPointerValue(authorization.Id)
	model.Org = types.StringPointerValue(authorization.Org)
	model.OrgID = types.StringPointerValue(authorization.OrgID)
	model.CreatedAt = rfc3339PointerValue(authorization.CreatedAt)
	model.UpdatedAt = rfc3339PointerValue(authorization.UpdatedAt)
	model.Description = types.StringPointerValue(authorization.Description)
	model.User = types.StringPointerValue(authorization.User)
	model.UserID = types.StringPointerValue(authorization.UserID)

	if authorization.Permissions != nil {
		model.Permissions = getPermissions(*authorization.Permissions)
	} else {
		model.Permissions = nil
	}
}
