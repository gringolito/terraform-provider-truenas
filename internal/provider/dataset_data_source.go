package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ datasource.DataSource = &DatasetDataSource{}

func NewDatasetDataSource() datasource.DataSource {
	return &DatasetDataSource{}
}

type DatasetDataSource struct {
	caller client.Caller
}

type DatasetDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	Path           types.String `tfsdk:"path"`
	Pool           types.String `tfsdk:"pool"`
	Name           types.String `tfsdk:"name"`
	Comments       types.String `tfsdk:"comments"`
	Compression    types.String `tfsdk:"compression"`
	Sync           types.String `tfsdk:"sync"`
	Atime          types.String `tfsdk:"atime"`
	Exec           types.String `tfsdk:"exec"`
	Readonly       types.String `tfsdk:"readonly"`
	Deduplication  types.String `tfsdk:"deduplication"`
	SnapDir        types.String `tfsdk:"snap_dir"`
	RecordSize     types.String `tfsdk:"record_size"`
	Quota          types.Int64  `tfsdk:"quota"`
	Refquota       types.Int64  `tfsdk:"refquota"`
	Reservation    types.Int64  `tfsdk:"reservation"`
	Refreservation types.Int64  `tfsdk:"refreservation"`
	Copies         types.Int64  `tfsdk:"copies"`
}

func (d *DatasetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (d *DatasetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a FILESYSTEM-type ZFS dataset by path. VOLUME-type datasets (zvols) are out of scope; encountering one is a clear, actionable error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mirrors `path`; the dataset's sole identity.",
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Full ZFS path of the dataset, including pool name (e.g. `tank/projects/child`).",
			},
			"pool": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Pool name — the first segment of `path`.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last segment of `path`.",
			},
			"comments": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Comments or description for the dataset.",
			},
			"compression": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Compression algorithm applied to the dataset.",
			},
			"sync": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synchronous write behavior.",
			},
			"atime": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether file access times are updated when files are accessed.",
			},
			"exec": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether files in this dataset can be executed.",
			},
			"readonly": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the dataset is read-only.",
			},
			"deduplication": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Deduplication setting for the dataset.",
			},
			"snap_dir": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Controls visibility of the `.zfs/snapshot` directory.",
			},
			"record_size": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Suggested block size for files in the dataset.",
			},
			"quota": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum disk space this dataset and its children can consume, in bytes.",
			},
			"refquota": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Maximum disk space this dataset itself can consume, in bytes.",
			},
			"reservation": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Minimum disk space guaranteed to this dataset and its children, in bytes.",
			},
			"refreservation": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Minimum disk space guaranteed to this dataset itself, in bytes.",
			},
			"copies": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Number of copies of data blocks maintained for redundancy.",
			},
		},
	}
}

func (d *DatasetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Caller)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected client.Caller, got %T", req.ProviderData))
		return
	}
	d.caller = c
}

func (d *DatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DatasetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path := config.Path.ValueString()
	if path == "" {
		resp.Diagnostics.AddError("Invalid configuration", "`path` must be a non-empty string.")
		return
	}

	ds, err := truenas.PoolDatasetGetFilesystem(ctx, d.caller, path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading dataset", err.Error())
		return
	}

	var state DatasetDataSourceModel
	if err := datasetToDataSourceModel(ds, &state); err != nil {
		resp.Diagnostics.AddError("Error decoding dataset", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func datasetToDataSourceModel(ds *truenas.PoolDataset, m *DatasetDataSourceModel) error {
	props, err := truenas.ExtractPoolDatasetProperties(ds)
	if err != nil {
		return fmt.Errorf("extracting dataset properties: %w", err)
	}

	pool, name := truenas.PoolDatasetPathParts(ds.Id)

	m.Id = types.StringValue(ds.Id)
	m.Path = types.StringValue(ds.Id)
	m.Pool = types.StringValue(pool)
	m.Name = types.StringValue(name)

	m.Comments = stringPtrToValue(props.Comments)
	m.Compression = stringPtrToValue(props.Compression)
	m.Sync = stringPtrToValue(props.Sync)
	m.Atime = stringPtrToValue(props.Atime)
	m.Exec = stringPtrToValue(props.Exec)
	m.Readonly = stringPtrToValue(props.Readonly)
	m.Deduplication = stringPtrToValue(props.Deduplication)
	m.SnapDir = stringPtrToValue(props.SnapDir)
	m.RecordSize = stringPtrToValue(props.RecordSize)

	m.Quota = int64PtrToValue(props.Quota)
	m.Refquota = int64PtrToValue(props.Refquota)
	m.Reservation = int64PtrToValue(props.Reservation)
	m.Refreservation = int64PtrToValue(props.Refreservation)
	m.Copies = int64PtrToValue(props.Copies)

	return nil
}
