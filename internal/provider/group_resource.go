package provider

import (
	"context"
	"fmt"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type groupResource struct {
	client *client.Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &groupResource{}

// NewGroupResource returns a new instance of the group resource.
func NewGroupResource() resource.Resource {
	return &groupResource{}
}

// Metadata sets the resource type name for the TrueNAS group resource.
func (r *groupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema defines the schema for the TrueNAS group resource.
func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use this resource to create a new group in the TrueNAS server.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "This is the API identifier for the group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"gid": schema.Int64Attribute{
				MarkdownDescription: "The Group ID (GID) is a unique number used to identify a Unix group. " +
					"Enter a number above 1000 for a group with user accounts. " +
					"If not set, TrueNAS will automatically assign with the next one available.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the group.",
				Required:            true,
			},
			"sudo_commands": schema.ListAttribute{
				MarkdownDescription: "A list of commands that group members may execute with elevated privileges. " +
					"User is prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				MarkdownDescription: "A list of commands that group members may execute with elevated privileges. " +
					"User is not prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"smb": schema.BoolAttribute{
				MarkdownDescription: "If set to True, the group can be used for SMB share ACL entries. " +
					"The group is mapped to an NT group account on the TrueNAS SMB server and has a sid value.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"userns_idmap": schema.DynamicAttribute{
				MarkdownDescription: "Specifies the subgid mapping for this group. " +
					"If DIRECT then the GID will be directly mapped to all containers. " +
					"Alternatively, the target GID may be explicitly specified. " +
					"If not set, then the GID will not be mapped.",
				Optional: true,
				Computed: true,
				Default:  dynamicdefault.StaticValue(types.DynamicValue(types.DynamicNull())),
			},
			"users": schema.ListAttribute{
				MarkdownDescription: "A list a user IDs (UIDs) for local users who are members of this group.",
				ElementType:         types.Int64Type,
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(types.Int64Type, []attr.Value{})),
			},
			"group": schema.StringAttribute{
				MarkdownDescription: "Human-readable string to identify the group. Identical to the name key.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"builtin": schema.BoolAttribute{
				MarkdownDescription: "If True, the group is an internal system account for the TrueNAS server.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"local": schema.BoolAttribute{
				MarkdownDescription: "If True, the group is local to the TrueNAS server. " +
					"If False, the group is provided by a directory service.",
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"sid": schema.StringAttribute{
				MarkdownDescription: "The Security Identifier (SID) of the user if the account an smb account. " +
					"The SMB server uses this value to check share access and for other purposes.",
				Computed: true,
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: "List of roles assigned to this groups. " +
					"Roles control administrative access to TrueNAS through the web UI and API.",
				ElementType: types.StringType,
				Computed:    true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure sets up the TrueNAS API client for the data source.
func (d *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	api, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf(
				"Expected *client.Client, got: %T. Please report this issue to the provider developers.",
				req.ProviderData,
			),
		)

		return
	}

	d.client = api
}

// Create handles the creation of the TrueNAS group resource.
func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan TrueNASGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var id int64
	service := NewTrueNASGroupService(ctx, r.client)
	resp.Diagnostics.Append(service.Create(plan, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS group was successfully created on the TrueNAS API")

	// Save data into Terraform state
	var state TrueNASGroupModel
	resp.Diagnostics.Append(service.Get(id, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, "Successfully created TrueNAS group resource")
}

// Read fetches the current state of the TrueNAS group resource.
func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform prior state data into the model
	var state TrueNASGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var id int64
	resp.Diagnostics.Append(state.GetID(&id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	var updated TrueNASGroupModel
	service := NewTrueNASGroupService(ctx, r.client)
	resp.Diagnostics.Append(service.Get(id, &updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS group was successfully read from TrueNAS API")

	resp.Diagnostics.Append(resp.State.Set(ctx, updated)...)

	tflog.Debug(ctx, "Successfully read TrueNAS group resource")
}

// Update handles updates to the TrueNAS group resource.
func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform data plan into the model
	var plan TrueNASGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var id int64
	service := NewTrueNASGroupService(ctx, r.client)
	resp.Diagnostics.Append(service.Update(plan, &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS group was successfully updated on the TrueNAS API")

	// Save data into Terraform state
	var state TrueNASGroupModel
	resp.Diagnostics.Append(service.Get(id, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, "Successfully created TrueNAS group resource")
}

// Delete removes the TrueNAS group resource.
func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state TrueNASGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service := NewTrueNASGroupService(ctx, r.client)
	resp.Diagnostics.Append(service.Delete(state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Successfully deleted TrueNAS group resource")
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
