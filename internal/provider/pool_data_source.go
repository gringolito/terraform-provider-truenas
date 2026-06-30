package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ datasource.DataSource = &PoolDataSource{}

func NewPoolDataSource() datasource.DataSource {
	return &PoolDataSource{}
}

type PoolDataSource struct {
	caller client.Caller
}

var poolScanAttrTypes = map[string]attr.Type{
	"function":         types.StringType,
	"state":            types.StringType,
	"start_time":       types.StringType,
	"end_time":         types.StringType,
	"percentage":       types.Float64Type,
	"bytes_to_process": types.Int64Type,
	"bytes_processed":  types.Int64Type,
	"bytes_issued":     types.Int64Type,
	"errors":           types.Int64Type,
	"pause":            types.StringType,
	"total_secs_left":  types.Int64Type,
}

type PoolDataSourceModel struct {
	Id              types.Int64  `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Guid            types.String `tfsdk:"guid"`
	Status          types.String `tfsdk:"status"`
	Path            types.String `tfsdk:"path"`
	Healthy         types.Bool   `tfsdk:"healthy"`
	Warning         types.Bool   `tfsdk:"warning"`
	IsUpgraded      types.Bool   `tfsdk:"is_upgraded"`
	StatusCode      types.String `tfsdk:"status_code"`
	StatusDetail    types.String `tfsdk:"status_detail"`
	Size            types.Int64  `tfsdk:"size"`
	Allocated       types.Int64  `tfsdk:"allocated"`
	Free            types.Int64  `tfsdk:"free"`
	Freeing         types.Int64  `tfsdk:"freeing"`
	DedupTableSize  types.Int64  `tfsdk:"dedup_table_size"`
	DedupTableQuota types.String `tfsdk:"dedup_table_quota"`
	Fragmentation   types.String `tfsdk:"fragmentation"`
	Autotrim        types.String `tfsdk:"autotrim"`
	Scan            types.Object `tfsdk:"scan"`
}

func (d *PoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *PoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a TrueNAS storage pool by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Internal TrueNAS API identifier of the pool.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the pool.",
			},
			"guid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Globally unique identifier of the pool.",
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current status of the pool (e.g. `ONLINE`, `DEGRADED`).",
			},
			"path": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mount path of the pool.",
			},
			"healthy": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the pool is healthy.",
			},
			"warning": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the pool has warnings.",
			},
			"is_upgraded": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the pool has been upgraded to the latest ZFS feature flags.",
			},
			"status_code": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Machine-readable pool status code.",
			},
			"status_detail": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Human-readable pool status detail.",
			},
			"size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Total pool capacity in bytes.",
			},
			"allocated": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Allocated pool space in bytes.",
			},
			"free": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Free pool space in bytes.",
			},
			"freeing": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Space being freed in bytes.",
			},
			"dedup_table_size": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Size of the deduplication table in bytes.",
			},
			"dedup_table_quota": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Quota setting for the deduplication table.",
			},
			"fragmentation": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Pool fragmentation percentage as a string.",
			},
			"autotrim": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Autotrim setting (`on` or `off`).",
			},
			"scan": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Details of the last pool scan (scrub). Null if no scan has run.",
				Attributes: map[string]schema.Attribute{
					"function": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Type of scan (e.g. `SCRUB`).",
					},
					"state": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "State of the scan (e.g. `FINISHED`, `SCANNING`).",
					},
					"start_time": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Scan start time in RFC 3339 format.",
					},
					"end_time": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Scan end time in RFC 3339 format.",
					},
					"percentage": schema.Float64Attribute{
						Computed:            true,
						MarkdownDescription: "Percentage of scan completed.",
					},
					"bytes_to_process": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Total bytes to process.",
					},
					"bytes_processed": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Bytes processed so far.",
					},
					"bytes_issued": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Bytes issued so far.",
					},
					"errors": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Number of scan errors.",
					},
					"pause": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "Time the scan was paused, in RFC 3339 format. Null if not paused.",
					},
					"total_secs_left": schema.Int64Attribute{
						Computed:            true,
						MarkdownDescription: "Estimated seconds remaining in the scan. Null if not scanning.",
					},
				},
			},
		},
	}
}

func (d *PoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PoolDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError("Invalid configuration", "`name` must be a non-empty string.")
		return
	}

	pool, err := truenas.PoolGetByName(ctx, d.caller, name)
	if err != nil {
		resp.Diagnostics.AddError("Error reading pool", err.Error())
		return
	}

	var state PoolDataSourceModel
	poolToDataSourceModel(pool, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func poolToDataSourceModel(pool *truenas.PoolDetail, m *PoolDataSourceModel, diags *diag.Diagnostics) {
	m.Id = types.Int64Value(pool.Id)
	m.Name = types.StringValue(pool.Name)
	m.Guid = types.StringValue(pool.Guid)
	m.Status = types.StringValue(pool.Status)
	m.Path = types.StringValue(pool.Path)
	m.Healthy = types.BoolValue(pool.Healthy)
	m.Warning = types.BoolValue(pool.Warning)
	m.IsUpgraded = types.BoolValue(pool.IsUpgraded)
	m.Autotrim = types.StringValue(pool.Autotrim.Parsed)

	if pool.StatusCode != nil {
		m.StatusCode = types.StringValue(*pool.StatusCode)
	} else {
		m.StatusCode = types.StringNull()
	}
	if pool.StatusDetail != nil {
		m.StatusDetail = types.StringValue(*pool.StatusDetail)
	} else {
		m.StatusDetail = types.StringNull()
	}
	if pool.Size != nil {
		m.Size = types.Int64Value(*pool.Size)
	} else {
		m.Size = types.Int64Null()
	}
	if pool.Allocated != nil {
		m.Allocated = types.Int64Value(*pool.Allocated)
	} else {
		m.Allocated = types.Int64Null()
	}
	if pool.Free != nil {
		m.Free = types.Int64Value(*pool.Free)
	} else {
		m.Free = types.Int64Null()
	}
	if pool.Freeing != nil {
		m.Freeing = types.Int64Value(*pool.Freeing)
	} else {
		m.Freeing = types.Int64Null()
	}
	if pool.DedupTableSize != nil {
		m.DedupTableSize = types.Int64Value(*pool.DedupTableSize)
	} else {
		m.DedupTableSize = types.Int64Null()
	}
	if pool.DedupTableQuota != nil {
		m.DedupTableQuota = types.StringValue(*pool.DedupTableQuota)
	} else {
		m.DedupTableQuota = types.StringNull()
	}
	if pool.Fragmentation != nil {
		m.Fragmentation = types.StringValue(*pool.Fragmentation)
	} else {
		m.Fragmentation = types.StringNull()
	}

	if pool.Scan == nil {
		m.Scan = types.ObjectNull(poolScanAttrTypes)
		return
	}

	startTime := ""
	if !pool.Scan.StartTime.IsZero() {
		startTime = pool.Scan.StartTime.UTC().Format(time.RFC3339)
	}
	endTime := ""
	if !pool.Scan.EndTime.IsZero() {
		endTime = pool.Scan.EndTime.UTC().Format(time.RFC3339)
	}

	pauseVal := types.StringNull()
	if pool.Scan.Pause != nil && !pool.Scan.Pause.IsZero() {
		pauseVal = types.StringValue(pool.Scan.Pause.UTC().Format(time.RFC3339))
	}

	totalSecsLeftVal := types.Int64Null()
	if pool.Scan.TotalSecsLeft != nil {
		totalSecsLeftVal = types.Int64Value(*pool.Scan.TotalSecsLeft)
	}

	scanVals := map[string]attr.Value{
		"function":         types.StringValue(pool.Scan.Function),
		"state":            types.StringValue(pool.Scan.State),
		"start_time":       types.StringValue(startTime),
		"end_time":         types.StringValue(endTime),
		"percentage":       types.Float64Value(pool.Scan.Percentage),
		"bytes_to_process": types.Int64Value(pool.Scan.BytesToProcess),
		"bytes_processed":  types.Int64Value(pool.Scan.BytesProcessed),
		"bytes_issued":     types.Int64Value(pool.Scan.BytesIssued),
		"errors":           types.Int64Value(pool.Scan.Errors),
		"pause":            pauseVal,
		"total_secs_left":  totalSecsLeftVal,
	}

	var dg diag.Diagnostics
	m.Scan, dg = types.ObjectValue(poolScanAttrTypes, scanVals)
	diags.Append(dg...)
}
