package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatasetDataSourceModel describes the schema and state for the dataset data source.
type DatasetDataSourceModel struct {
	ID                    types.String           `tfsdk:"id"`
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

// datasetDataSource implements the Terraform data source for TrueNAS datasets.
type datasetDataSource struct{}

// NewDatasetDataSource returns a new instance of the dataset data source.
func NewDatasetDataSource() datasource.DataSource {
	return &datasetDataSource{}
}

// Metadata sets the data source type name for Terraform.
func (d *datasetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

// Schema returns the schema for the dataset data source.
func (d *datasetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                       schema.StringAttribute{Computed: true},
			"name":                     schema.StringAttribute{Computed: true},
			"type":                     schema.StringAttribute{Computed: true},
			"volsize":                  schema.Int64Attribute{Computed: true},
			"volblocksize":             schema.StringAttribute{Computed: true},
			"sparse":                   schema.BoolAttribute{Computed: true},
			"force_size":               schema.BoolAttribute{Computed: true},
			"comments":                 schema.StringAttribute{Computed: true},
			"sync":                     schema.StringAttribute{Computed: true},
			"compression":              schema.StringAttribute{Computed: true},
			"atime":                    schema.BoolAttribute{Computed: true},
			"exec":                     schema.BoolAttribute{Computed: true},
			"managedby":                schema.StringAttribute{Computed: true},
			"quota":                    schema.Int64Attribute{Computed: true},
			"quota_warning":            schema.Int64Attribute{Computed: true},
			"quota_critical":           schema.Int64Attribute{Computed: true},
			"refquota":                 schema.Int64Attribute{Computed: true},
			"refquota_warning":         schema.Int64Attribute{Computed: true},
			"refquota_critical":        schema.Int64Attribute{Computed: true},
			"reservation":              schema.Int64Attribute{Computed: true},
			"refreservation":           schema.Int64Attribute{Computed: true},
			"special_small_block_size": schema.Int64Attribute{Computed: true},
			"copies":                   schema.Int64Attribute{Computed: true},
			"snapdir":                  schema.StringAttribute{Computed: true},
			"deduplication":            schema.StringAttribute{Computed: true},
			"checksum":                 schema.StringAttribute{Computed: true},
			"readonly":                 schema.BoolAttribute{Computed: true},
			"recordsize":               schema.StringAttribute{Computed: true},
			"casesensitivity":          schema.StringAttribute{Computed: true},
			"aclmode":                  schema.StringAttribute{Computed: true},
			"acltype":                  schema.StringAttribute{Computed: true},
			"share_type":               schema.StringAttribute{Computed: true},
			"xattr":                    schema.StringAttribute{Computed: true},
			"encryption_options": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"generate_key": schema.BoolAttribute{Computed: true},
					"pbkdf2iters":  schema.Int64Attribute{Computed: true},
					"algorithm":    schema.StringAttribute{Computed: true},
					"passphrase":   schema.StringAttribute{Computed: true, Sensitive: true},
					"key":          schema.StringAttribute{Computed: true, Sensitive: true},
				},
				Computed: true,
			},
			"encryption":         schema.BoolAttribute{Computed: true},
			"inherit_encryption": schema.BoolAttribute{Computed: true},
		},
	}
}

// Read fetches the dataset data and saves it into Terraform state.
func (d *datasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Not yet implemented
}
