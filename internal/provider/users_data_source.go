package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &UsersDataSource{}
	_ datasource.DataSourceWithConfigure = &UsersDataSource{}
)

// NewUsersDataSource is a helper function to simplify the provider implementation.
func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// UsersDataSource is the data source implementation.
type UsersDataSource struct {
	client influxdb2.Client
}

// UsersDataSourceModel describes the data source data model.
type UsersDataSourceModel struct {
	Users []UserModel `tfsdk:"users"`
}

// Metadata returns the data source type name.
func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "List all users.",

		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The user ID.",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The user name.",
						},
						"org_id": schema.StringAttribute{
							Computed:    true,
							Description: "The organization ID that the user belongs to. Null if the user is not a member of any organization.",
						},
						"org_role": schema.StringAttribute{
							Computed:    true,
							Description: "The role of the user in the organization (`member` or `owner`). Null if the user is not a member of any organization.",
						},
						"password": schema.StringAttribute{
							Computed:    true,
							Description: "The password of the user. This will be always `null`.",
							Sensitive:   true,
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The status of a user.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state UsersDataSourceModel

	users, err := d.client.UsersAPI().GetUsers(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list users",
			err.Error(),
		)

		return
	}

	// Map response body to model
	for _, user := range *users {
		userState := UserModel{
			Id:     types.StringValue(*user.Id),
			Name:   types.StringValue(user.Name),
			Status: types.StringValue(string(*user.Status)),
		}

		// Get organization membership information
		orgID, orgRole, err := d.getUserOrgMembership(ctx, *user.Id)
		if err != nil {
			// Log warning but don't fail - organization membership is optional information
			resp.Diagnostics.AddWarning(
				"Unable to get organization membership for user",
				fmt.Sprintf("Could not get organization membership for user %s: %s", user.Name, err.Error()),
			)
			// Set null values when we can't get org info
			userState.OrgId = types.StringNull()
			userState.OrgRole = types.StringNull()
		} else {
			// Set organization information if user is a member of an organization
			if orgID != "" {
				userState.OrgId = types.StringValue(orgID)
				userState.OrgRole = types.StringValue(orgRole)
			} else {
				userState.OrgId = types.StringNull()
				userState.OrgRole = types.StringNull()
			}
		}

		state.Users = append(state.Users, userState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// getUserOrgMembership gets the organization membership information for a user.
func (d *UsersDataSource) getUserOrgMembership(ctx context.Context, userID string) (orgID string, orgRole string, err error) {
	// Get all organizations
	orgs, err := d.client.OrganizationsAPI().GetOrganizations(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get organizations: %w", err)
	}

	// Check each organization for user membership
	for _, org := range *orgs {
		// Check if user is an owner
		owners, err := d.client.OrganizationsAPI().GetOwnersWithID(ctx, *org.Id)
		if err == nil && owners != nil {
			for _, owner := range *owners {
				if *owner.Id == userID {
					return *org.Id, "owner", nil
				}
			}
		}

		// Check if user is a member
		members, err := d.client.OrganizationsAPI().GetMembersWithID(ctx, *org.Id)
		if err == nil && members != nil {
			for _, member := range *members {
				if *member.Id == userID {
					return *org.Id, "member", nil
				}
			}
		}
	}

	// User is not a member of any organization
	return "", "", nil
}
