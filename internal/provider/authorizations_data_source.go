package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &AuthorizationsDataSource{}
	_ datasource.DataSourceWithConfigure = &AuthorizationsDataSource{}
)

// NewAuthorizationsDataSource is a helper function to simplify the provider implementation.
func NewAuthorizationsDataSource() datasource.DataSource {
	return &AuthorizationsDataSource{}
}

// AuthorizationsDataSource is the data source implementation.
type AuthorizationsDataSource struct {
	client influxdb2.Client
}

// AuthorizationsDataSourceModel describes the data source data model.
type AuthorizationsDataSourceModel struct {
	Authorizations []AuthorizationModel `tfsdk:"authorizations"`
}

// Metadata returns the data source type name.
func (d *AuthorizationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorizations"
}

// Schema defines the schema for the data source.
func (d *AuthorizationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Lists all authorizations.",

		Attributes: map[string]schema.Attribute{
			"authorizations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
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
							Description: "An Organization name. Specifies the organization that owns the authorization.",
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
							CustomType:  timetypes.RFC3339Type{},
							Description: "Authorization creation date in RFC3339 format.",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "Last Authorization update date in RFC3339 format.",
						},
						"permissions": schema.SetNestedAttribute{
							Computed:    true,
							Description: "A set of permissions for an authorization.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *AuthorizationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *AuthorizationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state AuthorizationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readAuthorizations, err := d.client.AuthorizationsAPI().GetAuthorizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Authorizations",
			formatAPIError(err),
		)

		return
	}

	// Map response body to model
	for _, authorization := range *readAuthorizations {
		var authorizationState AuthorizationModel
		populateAuthorizationModel(&authorizationState, &authorization)
		authorizationState.Token = types.StringPointerValue(authorization.Token)

		if authorization.Status != nil {
			authorizationState.Status = types.StringValue(string(*authorization.Status))
		} else {
			authorizationState.Status = types.StringNull()
		}

		state.Authorizations = append(state.Authorizations, authorizationState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
