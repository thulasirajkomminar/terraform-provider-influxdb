package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// BucketModel maps InfluxDB bucket schema data.
type BucketModel struct {
	Id              types.String      `tfsdk:"id"`
	OrgID           types.String      `tfsdk:"org_id"`
	Type            types.String      `tfsdk:"type"`
	Description     types.String      `tfsdk:"description"`
	Name            types.String      `tfsdk:"name"`
	CreatedAt       timetypes.RFC3339 `tfsdk:"created_at"`
	UpdatedAt       timetypes.RFC3339 `tfsdk:"updated_at"`
	RetentionPeriod types.Int64       `tfsdk:"retention_period"` // buckets cannot have more than one retention rule at this time
}

// populateBucketModel fills the model from an API bucket, guarding every
// optional field so a partial response can never panic and cleared values
// reset to null instead of lingering in state.
func populateBucketModel(model *BucketModel, bucket *domain.Bucket) {
	model.Id = types.StringPointerValue(bucket.Id)
	model.OrgID = types.StringPointerValue(bucket.OrgID)
	model.Name = types.StringValue(bucket.Name)
	model.Description = types.StringPointerValue(bucket.Description)
	model.CreatedAt = rfc3339PointerValue(bucket.CreatedAt)
	model.UpdatedAt = rfc3339PointerValue(bucket.UpdatedAt)

	if bucket.Type != nil {
		model.Type = types.StringValue(string(*bucket.Type))
	} else {
		model.Type = types.StringNull()
	}

	// A bucket has at most one retention rule; no rule means infinite
	// retention, which the API models as 0.
	if len(bucket.RetentionRules) > 0 {
		model.RetentionPeriod = types.Int64Value(bucket.RetentionRules[0].EverySeconds)
	} else {
		model.RetentionPeriod = types.Int64Value(0)
	}
}
