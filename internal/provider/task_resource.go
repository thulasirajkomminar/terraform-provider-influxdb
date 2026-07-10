package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TaskResource{}
	_ resource.ResourceWithConfigure   = &TaskResource{}
	_ resource.ResourceWithImportState = &TaskResource{}
)

// NewTaskResource is a helper function to simplify the provider implementation.
func NewTaskResource() resource.Resource {
	return &TaskResource{}
}

// TaskResource defines the resource implementation.
type TaskResource struct {
	client influxdb2.Client
}

// Metadata returns the resource type name.
func (r *TaskResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_task"
}

// Schema defines the schema for the resource.
func (r *TaskResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a task using Flux scripts with task options.",

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
				Description: "The Cron expression that defines the schedule on which the task runs.",
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
				Required:    true,
				Description: "The [Flux script](https://docs.influxdata.com/influxdb/v2/process-data/get-started/#components-of-a-task) that the task executes.",
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
				Required:    true,
				Description: "The organization ID. Specifies the organization that owns the task.",
			},
			"owner_id": schema.StringAttribute{
				Computed:    true,
				Description: "The user ID. Specifies the owner of the task.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Optional:    true,
				Description: "The status of the task (`active` or `inactive`).",
				Default:     stringdefault.StaticString("active"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"active", "inactive"}...),
				},
			},
			"updated_at": schema.StringAttribute{
				Computed:    true,
				CustomType:  timetypes.RFC3339Type{},
				Description: "The timestamp when the task was last updated, in RFC3339 format.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *TaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TaskModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createTaskResponse, err := r.client.TasksAPI().CreateTaskByFlux(ctx, plan.Flux.ValueString(), plan.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating task",
			"Could not create task, unexpected error: "+formatAPIError(err),
		)

		return
	}

	// Map response body to schema and populate Computed attribute values
	plan = convertDomainTaskToModel(ctx, createTaskResponse)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *TaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state TaskModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get all task using FindTask with empty filter
	task, err := r.client.TasksAPI().GetTaskByID(ctx, state.Id.ValueString())
	if err != nil {
		// The task was deleted outside of Terraform: remove it from state so
		// Terraform plans a re-create.
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)

			return
		}

		resp.Diagnostics.AddError(
			"Error getting task",
			formatAPIError(err),
		)
		return
	}

	// Map response body to model
	state = convertDomainTaskToModel(ctx, task)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TaskModel
	var state TaskModel

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
	updateTask := domain.Task{
		Id:          state.Id.ValueString(), // Need to include the ID for updates
		Cron:        plan.Cron.ValueStringPointer(),
		Description: plan.Description.ValueStringPointer(),
		Every:       plan.Every.ValueStringPointer(),
		Flux:        plan.Flux.ValueString(),
		Name:        plan.Name.ValueString(),
		OrgID:       plan.OrgID.ValueString(),
		Status:      (*domain.TaskStatusType)(plan.Status.ValueStringPointer()),
	}

	// Update existing task
	apiResponse, err := r.client.TasksAPI().UpdateTask(ctx, &updateTask)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating task",
			"Could not update task, unexpected error: "+formatAPIError(err),
		)
		return
	}

	// Handle properties conversion based on the configuration
	plan = convertDomainTaskToModel(ctx, apiResponse)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TaskModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing task
	err := r.client.TasksAPI().DeleteTaskWithID(ctx, state.Id.ValueString())
	if err != nil && !isNotFoundError(err) {
		resp.Diagnostics.AddError(
			"Error deleting task",
			"Could not delete task, unexpected error: "+formatAPIError(err),
		)

		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *TaskResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if client := configureResourceClient(req, resp); client != nil {
		r.client = client
	}
}

func (r *TaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
