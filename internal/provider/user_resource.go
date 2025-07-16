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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type userResource struct {
	client *client.Client
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &userResource{}

// NewUserResource returns a new instance of the user resource.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// Metadata sets the resource type name for the TrueNAS user resource.
func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the TrueNAS user resource.
func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Use this resource to create a new user in the TrueNAS server.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "This is the API identifier for the user.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uid": schema.Int64Attribute{
				MarkdownDescription: "The User ID (UID) is a unique number used to identify a Unix user. " +
					"If not set, TrueNAS will automatically assign with the next one available.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "String used to uniquely identify the user on the server. " +
					"In order to be portable across systems, local user names must be composed of characters from " +
					"the POSIX portable filename character set (IEEE Std 1003.1-2024 section 3.265). " +
					"This means alphanumeric characters, hyphens, underscores, and periods. " +
					"Usernames also may not begin with a hyphen or a period.",
				Required: true,
			},
			"home": schema.StringAttribute{
				MarkdownDescription: "The local file system path for the user account's home directory. " +
					"Typically, this is required only if the account has shell access (local or SSH) to TrueNAS. " +
					"This is not required for accounts used only for SMB share access.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("/var/empty"),
			},
			"shell": schema.StringAttribute{
				MarkdownDescription: "The user local shell interpreter.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/usr/bin/zsh"),
			},
			"full_name": schema.StringAttribute{
				MarkdownDescription: "Comment field to provide additional information about the user account. " +
					"Typically, this is the full name of the user or a short description of a service account. " +
					"There are no character set restrictions for this field. " +
					"This field is for information only. ",
				Required: true,
			},
			"smb": schema.BoolAttribute{
				MarkdownDescription: "The user account may be used to access SMB shares. " +
					"If set to true then TrueNAS stores an NT hash of the user account's password for local accounts. " +
					"This feature is unavailable for local accounts when General Purpose OS STIG compatibility mode is enabled. " +
					"If set to true the user is automatically added to the builtin_users group.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"userns_idmap": schema.DynamicAttribute{
				MarkdownDescription: "Specifies the subuid mapping for this user. " +
					"If DIRECT then the UID will be directly mapped to all containers. " +
					"Alternatively, the target UID may be explicitly specified. " +
					"If null, then the UID will not be mapped.",
				Optional: true,
				Computed: true,
				Default:  dynamicdefault.StaticValue(types.DynamicValue(types.DynamicNull())),
			},
			"group": schema.Int64Attribute{
				MarkdownDescription: "The Unix group GID for the user's primary group. " +
					"This is required if group_create is false.",
				Optional: true,
				Computed: true,
			},
			"groups": schema.ListAttribute{
				MarkdownDescription: "List of additional groups to which the user belongs.",
				Optional:            true,
				ElementType:         types.Int64Type,
			},
			"password_disabled": schema.BoolAttribute{
				MarkdownDescription: "If set to true password authentication for the user account is disabled. " +
					"Users with password authentication disabled may still authenticate to the TrueNAS server by other methods, such as SSH key-based authentication. " +
					"Password authentication is required for smb users.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"ssh_password_enabled": schema.BoolAttribute{
				MarkdownDescription: "Allow the user to authenticate to the TrueNAS SSH server using a password. " +
					"The established best practice is to use only key-based authentication for SSH servers.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"sshpubkey": schema.StringAttribute{
				MarkdownDescription: "SSH public keys corresponding to private keys that authenticate this user to the TrueNAS SSH server.",
				Optional:            true,
			},
			"locked": schema.BoolAttribute{
				MarkdownDescription: "If set to true the account is locked. " +
					"The account cannot be used to authenticate to the TrueNAS server.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"sudo_commands": schema.ListAttribute{
				MarkdownDescription: "A list of commands the user may execute with elevated privileges. " +
					"User is prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"sudo_commands_nopasswd": schema.ListAttribute{
				MarkdownDescription: "A list of commands the user may execute with elevated privileges. " +
					"User is not prompted for password when executing any command from the list.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "Email address of the user.",
				Optional:            true,
			},
			// State-only fields
			"group_create": schema.BoolAttribute{
				MarkdownDescription: "If set to true, the TrueNAS server automatically creates a new local group as the user's primary group.",
				Optional:            true,
			},
			"home_create": schema.BoolAttribute{
				MarkdownDescription: "If set to true, the TrueNAS server automatically creates a new home directory for the user in the specified home path.",
				Optional:            true,
			},
			"home_mode": schema.StringAttribute{
				MarkdownDescription: "Filesystem permission to set on the user's home directory.",
				Optional:            true,
			},
			"random_password": schema.BoolAttribute{
				MarkdownDescription: "If set to true, the TrueNAS server automatically generates a random 20 character password for the user.",
				Optional:            true,
			},
			// Write-only fields
			"password": schema.StringAttribute{
				MarkdownDescription: "The password for the user account. This is required if random_password is not set.",
				Optional:            true,
				Sensitive:           true,
				WriteOnly:           true,
			},
			// Read-only fields
			"builtin": schema.BoolAttribute{
				MarkdownDescription: "If true, the user account is an internal system account for the TrueNAS server.",
				Computed:            true,
			},
			"local": schema.BoolAttribute{
				MarkdownDescription: "If true, the account is local to the TrueNAS server. " +
					"If false, the account is provided by a directory service.",
				Computed: true,
			},
			"immutable": schema.BoolAttribute{
				MarkdownDescription: "If true, the account is system-provided and most fields related to it may not be changed.",
				Computed:            true,
			},
			"twofactor_auth_configured": schema.BoolAttribute{
				MarkdownDescription: "If true, the account has been configured for two-factor authentication. " +
					"Users are prompted for a second factor when authenticating to the TrueNAS web UI and API. " +
					"They may also be prompted when signing in to the TrueNAS SSH server using a password (depending on global two-factor authentication settings).",
				Computed: true,
			},
			"sid": schema.StringAttribute{
				MarkdownDescription: "The Security Identifier (SID) of the user if the account an smb account. " +
					"The SMB server uses this value to check share access and for other purposes.",
				Computed: true,
			},
			"last_password_change": schema.DynamicAttribute{
				MarkdownDescription: "The date of the last password change for local user accounts.",
				Computed:            true,
			},
			"password_age": schema.Int64Attribute{
				MarkdownDescription: "The age in days of the password for local user accounts.",
				Computed:            true,
			},
			"password_change_required": schema.BoolAttribute{
				MarkdownDescription: "Password change for local user account is required on next login.",
				Computed:            true,
			},
			"roles": schema.ListAttribute{
				MarkdownDescription: "List of roles assigned to this user's groups. " +
					"Roles control administrative access to TrueNAS through the web UI and API.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"api_keys": schema.ListAttribute{
				MarkdownDescription: "The IDs of the existing API keys for the user.",
				ElementType:         types.Int64Type,
				Computed:            true,
			},
		},
	}
}

// Configure sets up the TrueNAS API client for the reource.
func (d *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create handles the creation of the TrueNAS user resource.
func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Read Terraform plan data into the model
	var plan TrueNASUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var user TrueNASUserModel
	service := NewTrueNASUserService(ctx, r.client)
	resp.Diagnostics.Append(service.Create(plan, &user)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS user was successfully created on the TrueNAS API")

	// Save data into Terraform state
	state := TrueNASUserResourceModel{TrueNASUserModel: user}
	state.CopyFromPlan(plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, "Successfully created TrueNAS user resource")
}

// Read fetches the current state of the TrueNAS user resource.
func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Read Terraform prior state data into the model
	var state TrueNASUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var id int64
	resp.Diagnostics.Append(state.GetID(&id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var read TrueNASUserModel
	service := NewTrueNASUserService(ctx, r.client)
	resp.Diagnostics.Append(service.Get(id, &read)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS user was successfully read from TrueNAS API")

	// Save updated data into Terraform state
	updated := TrueNASUserResourceModel{TrueNASUserModel: read}
	updated.CopyFromState(state)
	resp.Diagnostics.Append(resp.State.Set(ctx, updated)...)

	tflog.Debug(ctx, "Successfully read TrueNAS user resource")
}

// Update handles updates to the TrueNAS user resource.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Read Terraform data plan into the model
	var plan TrueNASUserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var updated TrueNASUserModel
	service := NewTrueNASUserService(ctx, r.client)
	resp.Diagnostics.Append(service.Update(plan, &updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "TrueNAS user was successfully updated on the TrueNAS API")

	// Save data into Terraform state
	state := TrueNASUserResourceModel{TrueNASUserModel: updated}
	state.CopyFromPlan(plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Debug(ctx, "Successfully updated TrueNAS user resource")
}

// Delete removes the TrueNAS user resource.
func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Read Terraform prior state data into the model
	var state TrueNASUserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service := NewTrueNASUserService(ctx, r.client)
	resp.Diagnostics.Append(service.Delete(state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Successfully deleted TrueNAS user resource")
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
