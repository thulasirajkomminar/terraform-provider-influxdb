package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &UserResource{}
}

// UserResource defines the resource implementation.
type UserResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the resource.
func (r *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a user with optional organization membership. Supports adding users as members or owners of organizations.\n\n**Note:** InfluxDB Cloud doesn't let you manage user passwords through the API. Use the InfluxDB Cloud user interface (UI) to update your password.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The user ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The user name.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Description: "The password to set for the user.",
				Sensitive:   true,
			},
			"org_id": schema.StringAttribute{
				Optional:    true,
				Description: "The organization ID to add the user to. Required when `org_role` is specified.",
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("org_role")),
				},
			},
			"org_role": schema.StringAttribute{
				Optional:    true,
				Description: "The role of the user in the organization (`member` or `owner`).",
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"member", "owner"}...),
					stringvalidator.AlsoRequires(path.MatchRoot("org_id")),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The status of a user. Default: `active`",
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"active", "inactive"}...),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	status := domain.UserStatus(plan.Status.ValueString())
	createUser := domain.User{
		Name:   plan.Name.ValueString(),
		Status: &status,
	}

	// Convert properties map to domain format if provided
	createUserResponse, err := r.client.UsersAPI().CreateUser(ctx, &createUser)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating user",
			"Could not create user, unexpected error: "+err.Error(),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringPointerValue(createUserResponse.Id)
	plan.Name = types.StringValue(createUserResponse.Name)
	plan.Status = types.StringValue(string(*createUserResponse.Status))

	// Update the user with the password
	err = r.client.UsersAPI().UpdateUserPasswordWithID(ctx, plan.Id.ValueString(), plan.Password.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error setting user password",
			"Could not set user password, unexpected error: "+err.Error(),
		)
		return
	}
	// Update the plan with the password value
	plan.Password = types.StringValue(plan.Password.ValueString())

	// Handle organization membership if specified
	if !plan.OrgRole.IsNull() && !plan.OrgRole.IsUnknown() {
		orgRole := plan.OrgRole.ValueString()
		err = r.manageOrgMembership(ctx, plan.Id.ValueString(), "", plan.OrgId.ValueString(), "", orgRole)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error managing organization membership",
				"Could not manage organization membership, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state UserModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed user value from InfluxDB
	user, err := r.client.UsersAPI().FindUserByID(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"User not found",
			err.Error(),
		)

		return
	}

	// Overwrite items with refreshed state
	state = UserModel{
		Id:       types.StringPointerValue(user.Id),
		Name:     types.StringValue(user.Name),
		Status:   types.StringValue(string(*user.Status)),
		Password: state.Password, // Preserve password from current state since API doesn't return it
		OrgId:    state.OrgId,    // Preserve org_id from current state
		OrgRole:  state.OrgRole,  // Preserve org_role from current state
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserModel
	var state UserModel

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
	status := domain.UserStatus(plan.Status.ValueString())
	updateUser := domain.User{
		Id:     plan.Id.ValueStringPointer(),
		Name:   plan.Name.ValueString(),
		Status: &status,
	}

	// Update existing user
	apiResponse, err := r.client.UsersAPI().UpdateUser(ctx, &updateUser)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating user",
			"Could not update user, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.Id = types.StringPointerValue(apiResponse.Id)
	plan.Name = types.StringValue(apiResponse.Name)
	plan.Status = types.StringValue(string(*apiResponse.Status))

	// If password has changed, update the user password
	if !plan.Password.Equal(state.Password) {
		err = r.client.UsersAPI().UpdateUserPasswordWithID(ctx, plan.Id.ValueString(), plan.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating user password",
				"Could not update user password, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Set the password in the plan (it's always present since it's required)
	plan.Password = types.StringValue(plan.Password.ValueString())

	// Handle organization membership changes
	oldOrgId := ""
	if !state.OrgId.IsNull() {
		oldOrgId = state.OrgId.ValueString()
	}
	newOrgId := ""
	if !plan.OrgId.IsNull() && !plan.OrgId.IsUnknown() {
		newOrgId = plan.OrgId.ValueString()
	}
	oldRole := ""
	if !state.OrgRole.IsNull() {
		oldRole = state.OrgRole.ValueString()
	}
	newRole := ""
	if !plan.OrgRole.IsNull() && !plan.OrgRole.IsUnknown() {
		newRole = plan.OrgRole.ValueString()
	}

	// Only manage organization membership if there are changes
	if oldOrgId != newOrgId || oldRole != newRole {
		err = r.manageOrgMembership(ctx, plan.Id.ValueString(), oldOrgId, newOrgId, oldRole, newRole)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error managing organization membership",
				"Could not manage organization membership, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove user from organization if they were a member
	if !state.OrgId.IsNull() && !state.OrgRole.IsNull() {
		orgRole := state.OrgRole.ValueString()
		err := r.manageOrgMembership(ctx, *state.Id.ValueStringPointer(), state.OrgId.ValueString(), "", orgRole, "")
		if err != nil {
			// Log warning but don't fail deletion - the user will be deleted anyway
			resp.Diagnostics.AddWarning(
				"Warning removing user from organization",
				"Could not remove user from organization before deletion: "+err.Error(),
			)
		}
	}

	// Delete existing user
	err := r.client.UsersAPI().DeleteUserWithID(ctx, *state.Id.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting user",
			"Could not delete user, unexpected error: "+err.Error(),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// manageOrgMembership handles adding/removing/updating user membership in organizations.
func (r *UserResource) manageOrgMembership(ctx context.Context, userID string, oldOrgID, newOrgID, oldRole, newRole string) error {
	// Remove from old organization if it exists and is different from new one
	if oldOrgID != "" && oldOrgID != newOrgID {
		switch oldRole {
		case "owner":
			err := r.client.OrganizationsAPI().RemoveOwnerWithID(ctx, oldOrgID, userID)
			if err != nil {
				// Don't fail if user is not found in organization (might already be removed)
				if !isNotFoundError(err) {
					return fmt.Errorf("failed to remove user as owner from organization %s: %w", oldOrgID, err)
				}
			}
		case "member":
			err := r.client.OrganizationsAPI().RemoveMemberWithID(ctx, oldOrgID, userID)
			if err != nil {
				// Don't fail if user is not found in organization (might already be removed)
				if !isNotFoundError(err) {
					return fmt.Errorf("failed to remove user as member from organization %s: %w", oldOrgID, err)
				}
			}
		}
	}

	// Add to new organization if specified
	if newOrgID != "" {
		// If same org but role changed, remove old role first
		if oldOrgID == newOrgID && oldRole != newRole && oldRole != "" {
			switch oldRole {
			case "owner":
				err := r.client.OrganizationsAPI().RemoveOwnerWithID(ctx, oldOrgID, userID)
				if err != nil {
					// Don't fail if user is not found in organization (might already be removed)
					if !isNotFoundError(err) {
						return fmt.Errorf("failed to remove user as owner from organization %s: %w", oldOrgID, err)
					}
				}
			case "member":
				err := r.client.OrganizationsAPI().RemoveMemberWithID(ctx, oldOrgID, userID)
				if err != nil {
					// Don't fail if user is not found in organization (might already be removed)
					if !isNotFoundError(err) {
						return fmt.Errorf("failed to remove user as member from organization %s: %w", oldOrgID, err)
					}
				}
			}
		}

		// Add user to organization with new role (only if org changed or role changed)
		if oldOrgID != newOrgID || oldRole != newRole {
			switch newRole {
			case "owner":
				_, err := r.client.OrganizationsAPI().AddOwnerWithID(ctx, newOrgID, userID)
				if err != nil {
					return fmt.Errorf("failed to add user as owner to organization %s: %w", newOrgID, err)
				}
			case "member":
				_, err := r.client.OrganizationsAPI().AddMemberWithID(ctx, newOrgID, userID)
				if err != nil {
					return fmt.Errorf("failed to add user as member to organization %s: %w", newOrgID, err)
				}
			}
		}
	}

	return nil
}

// isNotFoundError checks if the error is a 404 not found error.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common "not found" error patterns
	errMsg := err.Error()
	return strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "404")
}
