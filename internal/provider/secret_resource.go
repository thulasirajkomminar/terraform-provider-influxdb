package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &SecretResource{}
	_ resource.ResourceWithConfigure = &SecretResource{}
)

// NewSecretResource is a helper function to simplify the provider implementation.
func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

// SecretResource defines the resource implementation.
type SecretResource struct {
	client influxdb2.Client
}

// SecretModel maps InfluxDB secret schema data.
type SecretModel struct {
	OrgID types.String `tfsdk:"org_id"`
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

// Metadata returns the resource type name.
func (r *SecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the resource.
func (r *SecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Creates and manages a secret in an organization's secret store. " +
			"**Drift limitation:** the InfluxDB API only lists secret keys and never returns secret values, " +
			"so Terraform can detect an out-of-band deletion of the key but cannot detect out-of-band changes " +
			"to the value. Import is not supported for the same reason.",

		Attributes: map[string]schema.Attribute{
			"org_id": schema.StringAttribute{
				Required:    true,
				Description: "The organization ID that owns the secret.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The secret key.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "The secret value. The API never returns this value, so out-of-band changes are not detected.",
			},
		},
	}
}

// upsertSecret creates or updates the secret key/value pair.
//
// The generated APIClient().PatchOrgsIDSecrets cannot be used here: its body
// parameter is a defined type over domain.Secrets, which does not inherit
// Secrets' custom MarshalJSON, and the underlying map is tagged `json:"-"` —
// so the request would serialize to an empty `{}` and silently upsert
// nothing. Instead, marshal domain.Secrets directly and send the PATCH
// through the client's HTTP service, which applies the same authentication
// and error handling.
func (r *SecretResource) upsertSecret(ctx context.Context, model SecretModel) error {
	body, err := json.Marshal(domain.Secrets{
		AdditionalProperties: map[string]string{
			model.Key.ValueString(): model.Value.ValueString(),
		},
	})
	if err != nil {
		return err
	}

	service := r.client.HTTPService()
	endpoint := service.ServerAPIURL() + "orgs/" + url.PathEscape(model.OrgID.ValueString()) + "/secrets"

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if apiErr := service.DoHTTPRequest(req, nil, nil); apiErr != nil {
		return apiErr
	}

	return nil
}

// Create creates the resource and sets the initial Terraform state.
func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsertSecret(ctx, plan); err != nil {
		resp.Diagnostics.AddError(
			"Error creating secret",
			"Could not create secret, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data. The API only
// returns secret keys, so this verifies the key still exists; the value in
// state is kept as-is because it can never be read back.
func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state SecretModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretKeys, err := r.client.APIClient().GetOrgsIDSecrets(ctx, &domain.GetOrgsIDSecretsAllParams{
		OrgID: state.OrgID.ValueString(),
	})
	if err != nil {
		// The owning organization was deleted outside of Terraform: the
		// secret is gone with it.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError(
			"Error getting secrets",
			formatAPIError(err),
		)

		return
	}

	found := false
	if secretKeys != nil && secretKeys.Secrets != nil {
		for _, key := range *secretKeys.Secrets {
			if key == state.Key.ValueString() {
				found = true
				break
			}
		}
	}

	// The secret was deleted outside of Terraform: remove it from state so
	// Terraform plans a re-create.
	if !found {
		resp.State.RemoveResource(ctx)

		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SecretModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.upsertSecret(ctx, plan); err != nil {
		resp.Diagnostics.AddError(
			"Error updating secret",
			"Could not update secret, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SecretModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing secret
	err := r.client.APIClient().DeleteOrgsIDSecretsID(ctx, &domain.DeleteOrgsIDSecretsIDAllParams{
		OrgID:    state.OrgID.ValueString(),
		SecretID: state.Key.ValueString(),
	})
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting secret",
			"Could not delete secret, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}
