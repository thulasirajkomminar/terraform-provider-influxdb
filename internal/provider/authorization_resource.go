package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &AuthorizationResource{}
	_ resource.ResourceWithImportState = &AuthorizationResource{}
	_ resource.ResourceWithImportState = &AuthorizationResource{}
)

// NewAuthorizationResource is a helper function to simplify the provider implementation.
func NewAuthorizationResource() resource.Resource {
	return &AuthorizationResource{}
}

// AuthorizationResource defines the resource implementation.
type AuthorizationResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *AuthorizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization"
}

// Schema defines the schema for the resource.
func (r *AuthorizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Creates and manages an authorization and returns the authorization with the generated API token. Use this resource to create/manage an authorization, which generates an API token with permissions to read or write to a specific resource or type of resource.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The authorization ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Description: "The API token.",
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Status of the token. Valid values are `active` or `inactive`.",
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"active", "inactive"}...),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "A description of the token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "An organization ID. Specifies the organization that owns the authorization.",
			},
			"org": schema.StringAttribute{
				Computed:    true,
				Description: "Organization name. Specifies the organization that owns the authorization.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "A user ID. Specifies the user that the authorization is scoped to.",
			},
			"user": schema.StringAttribute{
				Optional:    true,
				Description: "A user name. Specifies the user that the authorization is scoped to.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "Authorization creation date.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				Description: "Last Authorization update date.",
			},
			"permissions": schema.ListNestedAttribute{
				Required:    true,
				Description: "A list of permissions for an authorization.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Required:    true,
							Description: "Permission action. Valid values are `read` or `write`.",
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"read", "write"}...),
							},
						},
						"resource": schema.SingleNestedAttribute{
							Required: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Optional:    true,
									Description: "A resource ID. Identifies a specific resource.",
								},
								"name": schema.StringAttribute{
									Computed:    true,
									Description: "The name of the resource. **Note:** not all resource types have a name property.",
								},
								"org": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "An organization name. The organization that owns the resource.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"org_id": schema.StringAttribute{
									Computed:    true,
									Optional:    true,
									Description: "An organization ID. Identifies the organization that owns the resource.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"type": schema.StringAttribute{
									Required:    true,
									Description: "A resource type. Identifies the API resource's type (or kind).",
									Validators: []validator.String{
										stringvalidator.OneOf([]string{
											"authorizations",
											"buckets",
											"dashboards",
											"orgs",
											"tasks",
											"telegrafs",
											"users",
											"variables",
											"secrets",
											"labels",
											"views",
											"documents",
											"notificationRules",
											"notificationEndpoints",
											"checks",
											"dbrp",
											"annotations",
											"sources",
											"scrapers",
											"notebooks",
											"remotes",
											"replications",
											"instance",
											"flows",
											"functions",
											"subscriptions",
										}...),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *AuthorizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthorizationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var permissions []domain.Permission
	for _, permissionData := range plan.Permissions {
		permission := domain.Permission{
			Action: domain.PermissionAction(permissionData.Action.ValueString()),
			Resource: domain.Resource{
				Id:    permissionData.Resource.Id.ValueStringPointer(),
				Name:  permissionData.Resource.Name.ValueStringPointer(),
				Type:  domain.ResourceType(permissionData.Resource.Type.ValueString()),
				Org:   permissionData.Resource.Org.ValueStringPointer(),
				OrgID: permissionData.Resource.OrgID.ValueStringPointer(),
			},
		}

		permissions = append(permissions, permission)
	}

	createAuthorization := domain.Authorization{
		Id:          plan.Id.ValueStringPointer(),
		Org:         plan.Org.ValueStringPointer(),
		OrgID:       plan.OrgID.ValueStringPointer(),
		Permissions: &permissions,
		AuthorizationUpdateRequest: domain.AuthorizationUpdateRequest{
			Description: plan.Description.ValueStringPointer(),
		},
	}

	apiResponse, err := r.client.AuthorizationsAPI().CreateAuthorization(ctx, &createAuthorization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating authorization",
			"Could not create authorization, unexpected error: "+err.Error(),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringPointerValue(apiResponse.Id)
	plan.Org = types.StringPointerValue(apiResponse.Org)
	plan.OrgID = types.StringPointerValue(apiResponse.OrgID)
	plan.Token = types.StringPointerValue(apiResponse.Token)
	plan.CreatedAt = types.StringValue(apiResponse.CreatedAt.String())
	plan.UpdatedAt = types.StringValue(apiResponse.UpdatedAt.String())
	plan.Description = types.StringValue(*apiResponse.Description)
	plan.Permissions = getPermissions(*apiResponse.Permissions)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *AuthorizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state AuthorizationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed authorization value from InfluxDB
	readAuthorization, err := r.client.AuthorizationsAPI().GetAuthorizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Authorizations",
			err.Error(),
		)

		return
	}

	var authorization *domain.Authorization = nil
	for _, auth := range *readAuthorization {
		v := auth
		if *auth.Id == *state.Id.ValueStringPointer() {
			authorization = &v
			break
		}
	}

	if authorization == nil {
		resp.Diagnostics.AddError(
			"Authorization not found",
			"Authorization not found",
		)

		return
	}

	// Overwrite items with refreshed state
	state.Id = types.StringPointerValue(authorization.Id)
	state.Org = types.StringPointerValue(authorization.Org)
	state.OrgID = types.StringPointerValue(authorization.OrgID)
	state.CreatedAt = types.StringValue(authorization.CreatedAt.String())
	state.UpdatedAt = types.StringValue(authorization.UpdatedAt.String())
	state.Description = types.StringValue(*authorization.Description)
	state.Status = types.StringValue(string(*authorization.Status))
	state.Permissions = getPermissions(*authorization.Permissions)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *AuthorizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthorizationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	var status domain.AuthorizationUpdateRequestStatus
	if plan.Status.ValueString() == "active" {
		status = domain.AuthorizationUpdateRequestStatusActive
	} else {
		status = domain.AuthorizationUpdateRequestStatusInactive
	}

	// Update existing authorization
	apiResponse, err := r.client.AuthorizationsAPI().UpdateAuthorizationStatusWithID(ctx, *plan.Id.ValueStringPointer(), status)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating authorization",
			"Could not update authorization, unexpected error: "+err.Error(),
		)

		return
	}

	// Overwrite items with refreshed state
	plan.Id = types.StringPointerValue(apiResponse.Id)
	plan.Org = types.StringPointerValue(apiResponse.Org)
	plan.OrgID = types.StringPointerValue(apiResponse.OrgID)
	plan.Token = types.StringPointerValue(apiResponse.Token)
	plan.CreatedAt = types.StringValue(apiResponse.CreatedAt.String())
	plan.UpdatedAt = types.StringValue(apiResponse.UpdatedAt.String())
	plan.Description = types.StringValue(*apiResponse.Description)
	plan.Permissions = getPermissions(*apiResponse.Permissions)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *AuthorizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthorizationModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing authorization
	err := r.client.AuthorizationsAPI().DeleteAuthorizationWithID(ctx, *state.Id.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting authorization",
			"Could not delete authorization, unexpected error: "+err.Error(),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *AuthorizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(influxdb2.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb2.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *AuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

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
