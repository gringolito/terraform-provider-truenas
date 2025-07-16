package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EncryptionOptionsModel for nested encryption options
// Add enum types as needed for algorithm
type EncryptionOptionsModel struct {
	GenerateKey types.Bool   `tfsdk:"generate_key"`
	PBKDF2Iters types.Int64  `tfsdk:"pbkdf2iters"`
	Algorithm   types.String `tfsdk:"algorithm"`
	Passphrase  types.String `tfsdk:"passphrase"`
	Key         types.String `tfsdk:"key"`
}

// DatasetResourceModel maps resource schema data for the TrueNAS dataset resource.
type DatasetResourceModel struct {
	Name                  types.String           `tfsdk:"name"`
	Type                  types.String           `tfsdk:"type"`
	Volsize               types.Int64            `tfsdk:"volsize"`
	Volblocksize          types.String           `tfsdk:"volblocksize"`
	Sparse                types.Bool             `tfsdk:"sparse"`
	ForceSize             types.Bool             `tfsdk:"force_size"`
	Comments              types.String           `tfsdk:"comments"`
	Sync                  types.String           `tfsdk:"sync"`
	Compression           types.String           `tfsdk:"compression"`
	Atime                 types.Bool             `tfsdk:"atime"`
	Exec                  types.Bool             `tfsdk:"exec"`
	ManagedBy             types.String           `tfsdk:"managedby"`
	Quota                 types.Int64            `tfsdk:"quota"`
	QuotaWarning          types.Int64            `tfsdk:"quota_warning"`
	QuotaCritical         types.Int64            `tfsdk:"quota_critical"`
	Refquota              types.Int64            `tfsdk:"refquota"`
	RefquotaWarning       types.Int64            `tfsdk:"refquota_warning"`
	RefquotaCritical      types.Int64            `tfsdk:"refquota_critical"`
	Reservation           types.Int64            `tfsdk:"reservation"`
	Refreservation        types.Int64            `tfsdk:"refreservation"`
	SpecialSmallBlockSize types.Int64            `tfsdk:"special_small_block_size"`
	Copies                types.Int64            `tfsdk:"copies"`
	Snapdir               types.String           `tfsdk:"snapdir"`
	Deduplication         types.String           `tfsdk:"deduplication"`
	Checksum              types.String           `tfsdk:"checksum"`
	Readonly              types.Bool             `tfsdk:"readonly"`
	Recordsize            types.String           `tfsdk:"recordsize"`
	Casesensitivity       types.String           `tfsdk:"casesensitivity"`
	Aclmode               types.String           `tfsdk:"aclmode"`
	Acltype               types.String           `tfsdk:"acltype"`
	ShareType             types.String           `tfsdk:"share_type"`
	Xattr                 types.String           `tfsdk:"xattr"`
	EncryptionOptions     EncryptionOptionsModel `tfsdk:"encryption_options"`
	Encryption            types.Bool             `tfsdk:"encryption"`
	InheritEncryption     types.Bool             `tfsdk:"inherit_encryption"`
}

type datasetResource struct{}

// NewDatasetResource returns a new instance of the dataset resource.
func NewDatasetResource() resource.Resource {
	return &datasetResource{}
}

// Metadata sets the resource type name for the TrueNAS dataset resource.
func (r *datasetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

// Schema defines the schema for the TrueNAS dataset resource.
func (r *datasetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":                     schema.StringAttribute{Required: true},
			"type":                     schema.StringAttribute{Required: true},
			"volsize":                  schema.Int64Attribute{Optional: true},
			"volblocksize":             schema.StringAttribute{Optional: true},
			"sparse":                   schema.BoolAttribute{Optional: true},
			"force_size":               schema.BoolAttribute{Optional: true},
			"comments":                 schema.StringAttribute{Optional: true},
			"sync":                     schema.StringAttribute{Optional: true},
			"compression":              schema.StringAttribute{Optional: true},
			"atime":                    schema.BoolAttribute{Optional: true},
			"exec":                     schema.BoolAttribute{Optional: true},
			"managedby":                schema.StringAttribute{Optional: true},
			"quota":                    schema.Int64Attribute{Optional: true},
			"quota_warning":            schema.Int64Attribute{Optional: true},
			"quota_critical":           schema.Int64Attribute{Optional: true},
			"refquota":                 schema.Int64Attribute{Optional: true},
			"refquota_warning":         schema.Int64Attribute{Optional: true},
			"refquota_critical":        schema.Int64Attribute{Optional: true},
			"reservation":              schema.Int64Attribute{Optional: true},
			"refreservation":           schema.Int64Attribute{Optional: true},
			"special_small_block_size": schema.Int64Attribute{Optional: true},
			"copies":                   schema.Int64Attribute{Optional: true},
			"snapdir":                  schema.StringAttribute{Optional: true},
			"deduplication":            schema.StringAttribute{Optional: true},
			"checksum":                 schema.StringAttribute{Optional: true},
			"readonly":                 schema.BoolAttribute{Optional: true},
			"recordsize":               schema.StringAttribute{Optional: true},
			"casesensitivity":          schema.StringAttribute{Optional: true},
			"aclmode":                  schema.StringAttribute{Optional: true},
			"acltype":                  schema.StringAttribute{Optional: true},
			"share_type":               schema.StringAttribute{Optional: true},
			"xattr":                    schema.StringAttribute{Optional: true},
			"encryption_options": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"generate_key": schema.BoolAttribute{Optional: true},
					"pbkdf2iters":  schema.Int64Attribute{Optional: true},
					"algorithm":    schema.StringAttribute{Optional: true},
					"passphrase":   schema.StringAttribute{Optional: true, Sensitive: true},
					"key":          schema.StringAttribute{Optional: true, Sensitive: true},
				},
				Optional: true,
			},
			"encryption":         schema.BoolAttribute{Optional: true},
			"inherit_encryption": schema.BoolAttribute{Optional: true},
		},
	}
}

// Create handles the creation of the TrueNAS dataset resource.
func (r *datasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Not yet implemented
}

// Read fetches the current state of the TrueNAS dataset resource.
func (r *datasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Not yet implemented
}

// Update handles updates to the TrueNAS dataset resource.
func (r *datasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Not yet implemented
}

// Delete removes the TrueNAS dataset resource.
func (r *datasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Not yet implemented
}
