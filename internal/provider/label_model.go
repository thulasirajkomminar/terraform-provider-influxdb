package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// LabelModel maps InfluxDB label schema data.
type LabelModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	OrgID      types.String `tfsdk:"org_id"`
	Properties types.Map    `tfsdk:"properties"`
}

// convertLabelProperties converts domain.Label_Properties to types.Map
// Returns a null map if properties are nil/empty, otherwise converts AdditionalProperties.
func convertLabelProperties(ctx context.Context, props *domain.Label_Properties) (types.Map, diag.Diagnostics) {
	if props == nil || props.AdditionalProperties == nil {
		return types.MapNull(types.StringType), nil
	}

	return types.MapValueFrom(ctx, types.StringType, props.AdditionalProperties)
}
