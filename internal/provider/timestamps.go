package provider

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// rfc3339PointerValue converts an optional API timestamp to an RFC3339
// attribute value, returning null when the API did not return a value so
// stale timestamps never linger in state.
func rfc3339PointerValue(t *time.Time) timetypes.RFC3339 {
	if t == nil {
		return timetypes.NewRFC3339Null()
	}

	return timetypes.NewRFC3339TimeValue(*t)
}

// legacyTimestampLayouts are the formats produced by time.Time.String(),
// which prior provider versions stored in state for timestamp attributes.
var legacyTimestampLayouts = []string{
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
}

// normalizeLegacyTimestamp rewrites a timestamp stored by a prior provider
// version into RFC3339. Values that already parse as RFC3339 are returned
// unchanged, and unrecognized values are left untouched.
func normalizeLegacyTimestamp(raw string) string {
	if _, err := time.Parse(time.RFC3339, raw); err == nil {
		return raw
	}

	for _, layout := range legacyTimestampLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.Format(time.RFC3339)
		}
	}

	return raw
}

// timestampStateUpgrader returns a StateUpgrader that rewrites the given
// top-level string attributes from the legacy time.Time.String() format into
// RFC3339. Only the named attributes are touched; the rest of the state is
// passed through verbatim, so upgrading never alters resource identity.
func timestampStateUpgrader(attributes ...string) resource.StateUpgrader {
	return resource.StateUpgrader{
		StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
			if req.RawState == nil || len(req.RawState.JSON) == 0 {
				resp.Diagnostics.AddError(
					"Missing Prior State During Upgrade",
					"The prior state was empty while upgrading timestamp attributes to RFC3339. Please report this issue to the provider developers.",
				)

				return
			}

			var state map[string]any
			if err := json.Unmarshal(req.RawState.JSON, &state); err != nil {
				resp.Diagnostics.AddError(
					"Unable to Parse Prior State During Upgrade",
					"An unexpected error occurred while parsing the prior state to upgrade timestamp attributes to RFC3339: "+err.Error(),
				)

				return
			}

			for _, attribute := range attributes {
				if raw, ok := state[attribute].(string); ok && raw != "" {
					state[attribute] = normalizeLegacyTimestamp(raw)
				}
			}

			upgraded, err := json.Marshal(state)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Encode Upgraded State",
					"An unexpected error occurred while encoding the upgraded state: "+err.Error(),
				)

				return
			}

			resp.DynamicValue = &tfprotov6.DynamicValue{JSON: upgraded}
		},
	}
}
