package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &AuthorizationDataSource{}
	_ datasource.DataSourceWithConfigure = &AuthorizationDataSource{}
)

// NewAuthorizationDataSource is a helper function to simplify the provider implementation.
func NewAuthorizationDataSource() datasource.DataSource {
	return &AuthorizationDataSource{}
}

// AuthorizationsDataSource is the data source implementation.
type AuthorizationDataSource struct {
	client influxdb2.Client
}

// Metadata returns the data source type name.
func (d *AuthorizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization"
}

// Schema defines the schema for the data source.
func (d *AuthorizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves an authorization. Use this data source to retrieve information about an API token, including the token's permissions and the user that the token is scoped to.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The authorization ID.",
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Description: "The API token.",
				Sensitive:   true,
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the token.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "A description of the token.",
			},
			"org_id": schema.StringAttribute{
				Computed:    true,
				Description: "An organization ID. Specifies the organization that owns the authorization.",
			},
			"org": schema.StringAttribute{
				Computed:    true,
				Description: "Organization name. Specifies the organization that owns the authorization.",
			},
			"user_id": schema.StringAttribute{
				Computed:    true,
				Description: "A user ID. Specifies the user that the authorization is scoped to.",
			},
			"user": schema.StringAttribute{
				Computed:    true,
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
				Computed:    true,
				Description: "A list of permissions for an authorization.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Computed:    true,
							Description: "Permission action.",
						},
						"resource": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:    true,
									Description: "A resource ID. Identifies a specific resource.",
								},
								"name": schema.StringAttribute{
									Computed:    true,
									Description: "The name of the resource. **Note:** not all resource types have a name property.",
								},
								"org": schema.StringAttribute{
									Computed:    true,
									Description: "An organization name. The organization that owns the resource.",
								},
								"org_id": schema.StringAttribute{
									Computed:    true,
									Description: "An organization ID. Identifies the organization that owns the resource.",
								},
								"type": schema.StringAttribute{
									Computed:    true,
									Description: "A resource type. Identifies the API resource's type (or kind).",
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AuthorizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *AuthorizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AuthorizationModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readAuthorization, err := d.client.AuthorizationsAPI().GetAuthorizations(ctx)
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

	// Map response body to model
	for _, permissionData := range *authorization.Permissions {
		permissionState := AuthorizationPermissionModel{
			Action: types.StringValue(string(permissionData.Action)),
			Resource: AuthorizationPermissionResourceModel{
				Id:    types.StringPointerValue(permissionData.Resource.Id),
				Name:  types.StringPointerValue(permissionData.Resource.Name),
				Org:   types.StringPointerValue(permissionData.Resource.Org),
				OrgID: types.StringPointerValue(permissionData.Resource.OrgID),
				Type:  types.StringValue(string(permissionData.Resource.Type)),
			},
		}

		state.Permissions = append(state.Permissions, permissionState)
	}

	state.Id = types.StringPointerValue(authorization.Id)
	state.Org = types.StringPointerValue(authorization.Org)
	state.OrgID = types.StringPointerValue(authorization.OrgID)
	state.Token = types.StringPointerValue(authorization.Token)
	state.CreatedAt = types.StringValue(authorization.CreatedAt.String())
	state.UpdatedAt = types.StringValue(authorization.UpdatedAt.String())
	state.Description = types.StringValue(*authorization.Description)
	state.Status = types.StringValue(string(*authorization.Status))

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
