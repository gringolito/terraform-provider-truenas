package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ShareNFSDataSourceModel maps data source schema data for the TrueNAS NFS share data source.
type ShareNFSDataSourceModel struct {
	ID           types.Int64    `tfsdk:"id"`
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

// shareNFSDataSource implements the Terraform data source for TrueNAS NFS shares.
type shareNFSDataSource struct{}

// NewShareNFSDataSource returns a new instance of the NFS share data source.
func NewShareNFSDataSource() datasource.DataSource {
	return &shareNFSDataSource{}
}

// Metadata sets the data source type name for the TrueNAS NFS share data source.
func (d *shareNFSDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_nfs"
}

// Schema defines the schema for the TrueNAS NFS share data source.
func (d *shareNFSDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":            schema.Int64Attribute{Computed: true},
			"paths":         schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"comment":       schema.StringAttribute{Computed: true},
			"networks":      schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"hosts":         schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"alldirs":       schema.BoolAttribute{Computed: true},
			"ro":            schema.BoolAttribute{Computed: true},
			"quiet":         schema.BoolAttribute{Computed: true},
			"maproot_user":  schema.StringAttribute{Computed: true},
			"maproot_group": schema.StringAttribute{Computed: true},
			"mapall_user":   schema.StringAttribute{Computed: true},
			"mapall_group":  schema.StringAttribute{Computed: true},
			"security":      schema.ListAttribute{ElementType: types.StringType, Computed: true},
			"enabled":       schema.BoolAttribute{Computed: true},
		},
	}
}

// Read fetches the current state of the TrueNAS NFS share data source.
func (d *shareNFSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Not yet implemented
}
