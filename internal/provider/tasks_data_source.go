package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &TasksDataSource{}
	_ datasource.DataSourceWithConfigure = &TasksDataSource{}
)

// NewTasksDataSource is a helper function to simplify the provider implementation.
func NewTasksDataSource() datasource.DataSource {
	return &TasksDataSource{}
}

// TasksDataSource is the data source implementation.
type TasksDataSource struct {
	client influxdb2.Client
}

// TasksDataSourceModel describes the data source data model.
type TasksDataSourceModel struct {
	Tasks []TaskModel `tfsdk:"tasks"`
}

// Metadata returns the data source type name.
func (d *TasksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tasks"
}

// Schema defines the schema for the data source.
func (d *TasksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "List all tasks.",

		Attributes: map[string]schema.Attribute{
			"tasks": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"authorization_id": schema.StringAttribute{
							Computed:    true,
							Description: "The authorization ID. Specifies the authorization used when the task communicates with the query engine.",
						},
						"created_at": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "The timestamp when the task was created, in RFC3339 format.",
						},
						"cron": schema.StringAttribute{
							Computed:    true,
							Description: "The Cron expression that defines the schedule on which the task runs. InfluxDB uses the system time when evaluating Cron expressions.",
						},
						"description": schema.StringAttribute{
							Computed:    true,
							Description: "The description of the task.",
						},
						"every": schema.StringAttribute{
							Computed:    true,
							Description: "The interval [duration literal](https://docs.influxdata.com/influxdb/v2/reference/glossary/#rfc3339-timestamp) at which the task runs. every also determines when the task first runs, depending on the specified time.",
						},
						"flux": schema.StringAttribute{
							Computed:    true,
							Description: "The Flux script that the task executes.",
						},
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The task ID.",
						},
						"labels": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The labels associated with the task.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:    true,
										Description: "The label ID.",
									},
									"name": schema.StringAttribute{
										Computed:    true,
										Description: "The label name.",
									},
									"org_id": schema.StringAttribute{
										Computed:    true,
										Description: "The organization ID.",
									},
									"properties": schema.MapAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "The key-value pairs associated with this label.",
									},
								},
							},
						},
						"last_run_error": schema.StringAttribute{
							Computed:    true,
							Description: "The error message from the last task run, if any.",
						},
						"last_run_status": schema.StringAttribute{
							Computed:    true,
							Description: "The status of the last task run.",
						},
						"latest_completed": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "A timestamp [RFC3339 date/time format](https://docs.influxdata.com/influxdb/v2/reference/glossary/#rfc3339-timestamp) of the latest scheduled and completed run.",
						},
						"links": schema.SingleNestedAttribute{
							Computed:    true,
							Description: "Links related to the task.",
							Attributes: map[string]schema.Attribute{
								"labels": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
								"logs": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
								"members": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
								"owners": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
								"runs": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
								"self": schema.StringAttribute{
									Computed:    true,
									Description: "URI of resource.",
								},
							},
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The name of the task.",
						},
						"offset": schema.StringAttribute{
							Computed:    true,
							Description: "The duration to delay execution of the task after the scheduled time has elapsed. 0 removes the offset.",
						},
						"org": schema.StringAttribute{
							Computed:    true,
							Description: "The organization name. Specifies the organization that owns the task.",
						},
						"org_id": schema.StringAttribute{
							Computed:    true,
							Description: "The organization ID. Specifies the organization that owns the task.",
						},
						"owner_id": schema.StringAttribute{
							Computed:    true,
							Description: "The user ID. Specifies the owner of the task.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The status of the task (`active` or `inactive`).",
						},
						"updated_at": schema.StringAttribute{
							Computed:    true,
							CustomType:  timetypes.RFC3339Type{},
							Description: "The timestamp when the task was last updated, in RFC3339 format.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TasksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if client := configureDataSourceClient(req, resp); client != nil {
		d.client = client
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *TasksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TasksDataSourceModel

	// Get all tasks using FindTasks with empty filter
	tasks, err := d.client.TasksAPI().FindTasks(ctx, &api.TaskFilter{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list tasks",
			formatAPIError(err),
		)
		return
	}

	// Map response body to model
	for _, task := range tasks {
		taskState := convertDomainTaskToModel(ctx, &task)
		state.Tasks = append(state.Tasks, taskState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
