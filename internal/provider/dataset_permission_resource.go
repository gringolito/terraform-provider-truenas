package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatasetPermissionResourceModel describes the schema and state for the dataset permission resource.
type DatasetPermissionResourceModel struct {
	User      types.String   `tfsdk:"user"`
	Group     types.String   `tfsdk:"group"`
	Mode      types.String   `tfsdk:"mode"`
	ACL       []types.String `tfsdk:"acl"`
	StripACL  types.Bool     `tfsdk:"stripacl"`
	Recursive types.Bool     `tfsdk:"recursive"`
	Traverse  types.Bool     `tfsdk:"traverse"`
}

// datasetPermissionResource implements the Terraform resource for TrueNAS dataset permissions.
type datasetPermissionResource struct{}

// NewDatasetPermissionResource returns a new instance of the dataset permission resource.
func NewDatasetPermissionResource() resource.Resource {
	return &datasetPermissionResource{}
}

// Metadata sets the resource type name for Terraform.
func (r *datasetPermissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset_permission"
}

// Schema returns the schema for the dataset permission resource.
func (r *datasetPermissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"user":      schema.StringAttribute{Optional: true},
			"group":     schema.StringAttribute{Optional: true},
			"mode":      schema.StringAttribute{Optional: true},
			"acl":       schema.ListAttribute{ElementType: types.StringType, Optional: true},
			"stripacl":  schema.BoolAttribute{Optional: true},
			"recursive": schema.BoolAttribute{Optional: true},
			"traverse":  schema.BoolAttribute{Optional: true},
		},
	}
}

// Create creates the dataset permission resource in Terraform state.
func (r *datasetPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Not yet implemented
}

// Read fetches the dataset permission resource data and saves it into Terraform state.
func (r *datasetPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Not yet implemented
}

// Update updates the dataset permission resource in Terraform state.
func (r *datasetPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not yet implemented
}

// Delete removes the dataset permission resource from Terraform state.
func (r *datasetPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Not yet implemented
}
