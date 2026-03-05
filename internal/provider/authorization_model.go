package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

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
	CreatedAt   types.String                   `tfsdk:"created_at"`
	UpdatedAt   types.String                   `tfsdk:"updated_at"`
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
