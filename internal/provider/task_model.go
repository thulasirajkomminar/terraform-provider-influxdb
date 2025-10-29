package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// TaskModel maps InfluxDB label schema data.
type TaskModel struct {
	AuthorizationID types.String `tfsdk:"authorization_id"`
	CreatedAt       types.String `tfsdk:"created_at"`
	Cron            types.String `tfsdk:"cron"`
	Description     types.String `tfsdk:"description"`
	Every           types.String `tfsdk:"every"`
	Flux            types.String `tfsdk:"flux"`
	Id              types.String `tfsdk:"id"`
	Labels          types.List   `tfsdk:"labels"`
	LastRunError    types.String `tfsdk:"last_run_error"`
	LastRunStatus   types.String `tfsdk:"last_run_status"`
	LatestCompleted types.String `tfsdk:"latest_completed"`
	Links           types.Object `tfsdk:"links"`
	Name            types.String `tfsdk:"name"`
	Offset          types.String `tfsdk:"offset"`
	Org             types.String `tfsdk:"org"`
	OrgID           types.String `tfsdk:"org_id"`
	OwnerID         types.String `tfsdk:"owner_id"`
	Status          types.String `tfsdk:"status"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

type TaskLinksModel struct {
	Labels  types.String `tfsdk:"labels"`
	Logs    types.String `tfsdk:"logs"`
	Members types.String `tfsdk:"members"`
	Owners  types.String `tfsdk:"owners"`
	Runs    types.String `tfsdk:"runs"`
	Self    types.String `tfsdk:"self"`
}

// convertDomainTaskToModel converts a domain.Task to TaskModel.
func convertDomainTaskToModel(ctx context.Context, task *domain.Task) TaskModel {
	// Convert labels if present
	labelsList := convertLabelsToList(ctx, task.Labels)

	// Convert links if present
	linksObject := convertLinksToObject(task.Links)

	return TaskModel{
		AuthorizationID: types.StringPointerValue(task.AuthorizationID),
		CreatedAt:       convertTimeToString(task.CreatedAt),
		Cron:            types.StringPointerValue(task.Cron),
		Description:     types.StringPointerValue(task.Description),
		Every:           types.StringPointerValue(task.Every),
		Flux:            types.StringValue(task.Flux),
		Id:              types.StringValue(task.Id),
		Labels:          labelsList,
		LastRunError:    types.StringPointerValue(task.LastRunError),
		LastRunStatus:   convertTaskStatusToString(task.LastRunStatus),
		LatestCompleted: convertTimeToString(task.LatestCompleted),
		Links:           linksObject,
		Name:            types.StringValue(task.Name),
		Offset:          types.StringPointerValue(task.Offset),
		Org:             types.StringPointerValue(task.Org),
		OrgID:           types.StringValue(task.OrgID),
		OwnerID:         types.StringPointerValue(task.OwnerID),
		Status:          convertTaskStatusToString((*domain.TaskLastRunStatus)(task.Status)),
		UpdatedAt:       convertTimeToString(task.UpdatedAt),
	}
}

// Helper function to convert domain labels to types.List.
func convertLabelsToList(ctx context.Context, domainLabels *domain.Labels) types.List {
	if domainLabels == nil || len(*domainLabels) == 0 {
		// Return null list with proper element type
		labelObjectType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":         types.StringType,
				"name":       types.StringType,
				"org_id":     types.StringType,
				"properties": types.MapType{ElemType: types.StringType},
			},
		}
		return types.ListNull(labelObjectType)
	}

	labelObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":         types.StringType,
			"name":       types.StringType,
			"org_id":     types.StringType,
			"properties": types.MapType{ElemType: types.StringType},
		},
	}

	var labelValues []attr.Value
	for _, label := range *domainLabels {
		propertiesMap, diags := convertLabelProperties(ctx, label.Properties)
		if diags.HasError() {
			// If there's an error converting properties, use a null typed map
			propertiesMap = types.MapNull(types.StringType)
		}
		// Ensure the map is properly typed even if it's valid
		if propertiesMap.IsNull() {
			propertiesMap = types.MapNull(types.StringType)
		} else if propertiesMap.IsUnknown() {
			propertiesMap = types.MapUnknown(types.StringType)
		}

		labelObject, labelDiags := types.ObjectValue(
			labelObjectType.AttrTypes,
			map[string]attr.Value{
				"id":         types.StringPointerValue(label.Id),
				"name":       types.StringPointerValue(label.Name),
				"org_id":     types.StringPointerValue(label.OrgID),
				"properties": propertiesMap,
			},
		)
		if labelDiags.HasError() {
			// If there's an error creating the object, create a null object
			labelObject = types.ObjectNull(labelObjectType.AttrTypes)
		}
		labelValues = append(labelValues, labelObject)
	}

	labelsListValue, listDiags := types.ListValue(labelObjectType, labelValues)
	if listDiags.HasError() {
		// If there's an error creating the list, return null list
		return types.ListNull(labelObjectType)
	}

	return labelsListValue
}

// Helper function to convert domain.Links to types.Object.
func convertLinksToObject(links *struct {
	Labels  *domain.Link `json:"labels,omitempty"`
	Logs    *domain.Link `json:"logs,omitempty"`
	Members *domain.Link `json:"members,omitempty"`
	Owners  *domain.Link `json:"owners,omitempty"`
	Runs    *domain.Link `json:"runs,omitempty"`
	Self    *domain.Link `json:"self,omitempty"`
}) types.Object {
	if links == nil {
		// Return null object with proper attributes
		return types.ObjectNull(map[string]attr.Type{
			"labels":  types.StringType,
			"logs":    types.StringType,
			"members": types.StringType,
			"owners":  types.StringType,
			"runs":    types.StringType,
			"self":    types.StringType,
		})
	}

	attributeMap := map[string]attr.Value{
		"labels":  convertLinkToString(links.Labels),
		"logs":    convertLinkToString(links.Logs),
		"members": convertLinkToString(links.Members),
		"owners":  convertLinkToString(links.Owners),
		"runs":    convertLinkToString(links.Runs),
		"self":    convertLinkToString(links.Self),
	}

	attributeTypes := map[string]attr.Type{
		"labels":  types.StringType,
		"logs":    types.StringType,
		"members": types.StringType,
		"owners":  types.StringType,
		"runs":    types.StringType,
		"self":    types.StringType,
	}

	objectValue, diags := types.ObjectValue(attributeTypes, attributeMap)
	if diags.HasError() {
		// If there's an error, return null object
		return types.ObjectNull(attributeTypes)
	}

	return objectValue
}

// Helper function to convert domain.Link to string.
func convertLinkToString(link *domain.Link) types.String {
	if link != nil {
		return types.StringValue(string(*link))
	}
	return types.StringNull()
}

// Helper function to convert time.Time to string.
func convertTimeToString(t *time.Time) types.String {
	if t != nil {
		return types.StringValue(t.Format(time.RFC3339))
	}
	return types.StringNull()
}

// Helper function to convert TaskLastRunStatus to string.
func convertTaskStatusToString(status *domain.TaskLastRunStatus) types.String {
	if status != nil {
		return types.StringValue(string(*status))
	}
	return types.StringNull()
}
