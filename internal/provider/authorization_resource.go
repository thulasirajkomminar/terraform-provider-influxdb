package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                 = &AuthorizationResource{}
	_ resource.ResourceWithConfigure    = &AuthorizationResource{}
	_ resource.ResourceWithImportState  = &AuthorizationResource{}
	_ resource.ResourceWithUpgradeState = &AuthorizationResource{}
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

		// Version 1 stores timestamps as RFC3339 and permissions as a set.
		Version: 1,

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
				Computed:    true,
				Optional:    true,
				Description: "A user ID. Specifies the user that the authorization is scoped to.",
			},
			"user": schema.StringAttribute{
				Computed:    true,
				Description: "A user name. Specifies the user that the authorization is scoped to.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Authorization creation date in RFC3339 format.",
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "Last Authorization update date in RFC3339 format.",
			},
			"permissions": schema.SetNestedAttribute{
				Required:    true,
				Description: "A set of permissions for an authorization. The order of the permissions is not significant.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
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

// UpgradeState migrates state written by prior provider versions: timestamps
// move from Go's time.Time.String() format to RFC3339. The permissions
// list-to-set change needs no value rewrite.
func (r *AuthorizationResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: timestampStateUpgrader("created_at", "updated_at"),
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
	permissions := buildPermissions(plan.Permissions)

	createAuthorization := domain.Authorization{
		OrgID:       plan.OrgID.ValueStringPointer(),
		Permissions: &permissions,
		AuthorizationUpdateRequest: domain.AuthorizationUpdateRequest{
			Description: plan.Description.ValueStringPointer(),
		},
	}

	if !plan.UserID.IsNull() && !plan.UserID.IsUnknown() {
		createAuthorization.UserID = plan.UserID.ValueStringPointer()
	}

	apiResponse, err := r.client.AuthorizationsAPI().CreateAuthorization(ctx, &createAuthorization)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating authorization",
			"Could not create authorization, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	populateAuthorizationModel(&plan, apiResponse)
	plan.Token = types.StringPointerValue(apiResponse.Token)

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
			formatAPIError(err),
		)

		return
	}

	var authorization *domain.Authorization = nil
	for _, auth := range *readAuthorization {
		v := auth
		if v.Id != nil && *v.Id == state.Id.ValueString() {
			authorization = &v
			break
		}
	}

	// The authorization was deleted outside of Terraform: remove it from
	// state so Terraform plans a re-create.
	if authorization == nil {
		resp.State.RemoveResource(ctx)

		return
	}

	// Overwrite items with refreshed state
	populateAuthorizationModel(&state, authorization)

	if authorization.Status != nil {
		state.Status = types.StringValue(string(*authorization.Status))
	} else {
		state.Status = types.StringNull()
	}

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
	apiResponse, err := r.client.AuthorizationsAPI().UpdateAuthorizationStatusWithID(ctx, plan.Id.ValueString(), status)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating authorization",
			"Could not update authorization, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Overwrite items with refreshed state
	populateAuthorizationModel(&plan, apiResponse)

	if apiResponse.Token != nil {
		plan.Token = types.StringPointerValue(apiResponse.Token)
	}

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
	err := r.client.AuthorizationsAPI().DeleteAuthorizationWithID(ctx, state.Id.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting authorization",
			"Could not delete authorization, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *AuthorizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *AuthorizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
