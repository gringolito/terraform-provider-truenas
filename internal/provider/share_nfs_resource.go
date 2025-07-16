package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ShareNFSResourceModel maps resource schema data for the TrueNAS NFS share resource.
type ShareNFSResourceModel struct {
	Paths        []types.String `tfsdk:"paths"`
	Comment      types.String   `tfsdk:"comment"`
	Networks     []types.String `tfsdk:"networks"`
	Hosts        []types.String `tfsdk:"hosts"`
	AllDirs      types.Bool     `tfsdk:"alldirs"`
	RO           types.Bool     `tfsdk:"ro"`
	Quiet        types.Bool     `tfsdk:"quiet"`
	MaprootUser  types.String   `tfsdk:"maproot_user"`
	MaprootGroup types.String   `tfsdk:"maproot_group"`
	MapallUser   types.String   `tfsdk:"mapall_user"`
	MapallGroup  types.String   `tfsdk:"mapall_group"`
	Security     []types.String `tfsdk:"security"`
	Enabled      types.Bool     `tfsdk:"enabled"`
}

// shareNFSResource implements the Terraform resource for TrueNAS NFS shares.
type shareNFSResource struct{}

// NewShareNFSResource returns a new instance of the NFS share resource.
func NewShareNFSResource() resource.Resource {
	return &shareNFSResource{}
}

// Metadata sets the resource type name for the TrueNAS NFS share resource.
func (r *shareNFSResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_nfs"
}

// Schema defines the schema for the TrueNAS NFS share resource.
func (r *shareNFSResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"paths":         schema.ListAttribute{ElementType: types.StringType, Required: true},
			"comment":       schema.StringAttribute{Optional: true},
			"networks":      schema.ListAttribute{ElementType: types.StringType, Optional: true},
			"hosts":         schema.ListAttribute{ElementType: types.StringType, Optional: true},
			"alldirs":       schema.BoolAttribute{Optional: true},
			"ro":            schema.BoolAttribute{Optional: true},
			"quiet":         schema.BoolAttribute{Optional: true},
			"maproot_user":  schema.StringAttribute{Optional: true},
			"maproot_group": schema.StringAttribute{Optional: true},
			"mapall_user":   schema.StringAttribute{Optional: true},
			"mapall_group":  schema.StringAttribute{Optional: true},
			"security":      schema.ListAttribute{ElementType: types.StringType, Optional: true},
			"enabled":       schema.BoolAttribute{Optional: true},
		},
	}
}

// Create handles the creation of the TrueNAS NFS share resource.
func (r *shareNFSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Not yet implemented
}

// Read fetches the current state of the TrueNAS NFS share resource.
func (r *shareNFSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Not yet implemented
}

// Update handles updates to the TrueNAS NFS share resource.
func (r *shareNFSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not yet implemented
}

// Delete removes the TrueNAS NFS share resource.
func (r *shareNFSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Not yet implemented
}
