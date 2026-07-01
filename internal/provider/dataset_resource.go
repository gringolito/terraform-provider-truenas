package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/truenas"
)

var _ resource.Resource = &DatasetResource{}
var _ resource.ResourceWithImportState = &DatasetResource{}

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

type DatasetResource struct {
	caller client.Caller
}

type DatasetResourceModel struct {
	Id              types.String `tfsdk:"id"`
	Path            types.String `tfsdk:"path"`
	Pool            types.String `tfsdk:"pool"`
	Name            types.String `tfsdk:"name"`
	PreventDeletion types.Bool   `tfsdk:"prevent_deletion"`
	Comments        types.String `tfsdk:"comments"`
	Compression     types.String `tfsdk:"compression"`
	Sync            types.String `tfsdk:"sync"`
	Atime           types.String `tfsdk:"atime"`
	Exec            types.String `tfsdk:"exec"`
	Readonly        types.String `tfsdk:"readonly"`
	Deduplication   types.String `tfsdk:"deduplication"`
	SnapDir         types.String `tfsdk:"snap_dir"`
	RecordSize      types.String `tfsdk:"record_size"`
	Quota           types.Int64  `tfsdk:"quota"`
	Refquota        types.Int64  `tfsdk:"refquota"`
	Reservation     types.Int64  `tfsdk:"reservation"`
	Refreservation  types.Int64  `tfsdk:"refreservation"`
	Copies          types.Int64  `tfsdk:"copies"`
}

func (r *DatasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a FILESYSTEM-type ZFS dataset on TrueNAS SCALE. VOLUME-type datasets (zvols) are out of scope; encountering one is a clear, actionable error.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Mirrors `path`; the dataset's sole identity.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Full ZFS path of the dataset, including pool name (e.g. `tank/projects/child`). Changing this forces recreation.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"pool": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Pool name — the first segment of `path`. Derived, read-only.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Last segment of `path`. Derived, read-only.",
			},
			"prevent_deletion": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "When `true`, `Delete` refuses to destroy the dataset — including when destruction is " +
					"triggered by a `path` change forcing replacement. Defaults to `false`. To destroy a protected dataset, " +
					"first apply with this set to `false`.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"comments": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Comments or description for the dataset.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"compression": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Compression algorithm to use for the dataset (e.g. `LZ4`, `ZSTD`, `OFF`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sync": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Synchronous write behavior (`STANDARD`, `ALWAYS`, `DISABLED`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"atime": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether file access times are updated when files are accessed (`ON`, `OFF`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"exec": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether files in this dataset can be executed (`ON`, `OFF`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"readonly": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the dataset is read-only (`ON`, `OFF`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"deduplication": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Deduplication setting for the dataset (`ON`, `OFF`, `VERIFY`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"snap_dir": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Controls visibility of the `.zfs/snapshot` directory (`VISIBLE`, `HIDDEN`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"record_size": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Suggested block size for files in the dataset (e.g. `128K`, `1M`, `INHERIT`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"quota": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum disk space this dataset and its children can consume, in bytes. `0` means no quota.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"refquota": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Maximum disk space this dataset itself can consume, in bytes. `0` means no quota.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"reservation": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum disk space guaranteed to this dataset and its children, in bytes. `0` means none.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"refreservation": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum disk space guaranteed to this dataset itself, in bytes. `0` means none.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"copies": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of copies of data blocks to maintain for redundancy.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *DatasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(client.Caller)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected client.Caller, got %T", req.ProviderData))
		return
	}
	r.caller = c
}

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	defaultPreventDeletion(&plan)

	args := truenas.PoolDatasetCreateArgs{Name: plan.Path.ValueString()}
	datasetMutableFieldsFromModel(plan).applyToCreateArgs(&args)

	ds, err := truenas.PoolDatasetCreate(ctx, r.caller, args)
	if err != nil {
		resp.Diagnostics.AddError("Error creating dataset", err.Error())
		return
	}

	if err := datasetToModel(ds, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading dataset after create", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	defaultPreventDeletion(&state)

	ds, err := truenas.PoolDatasetGetFilesystem(ctx, r.caller, state.Path.ValueString())
	if err != nil {
		if isNotFoundErr(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading dataset", err.Error())
		return
	}

	if err := datasetToModel(ds, &state); err != nil {
		resp.Diagnostics.AddError("Error decoding dataset", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatasetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var args truenas.PoolDatasetUpdateArgs
	datasetMutableFieldsFromModel(plan).applyToUpdateArgs(&args)

	ds, err := truenas.PoolDatasetUpdate(ctx, r.caller, plan.Path.ValueString(), args)
	if err != nil {
		resp.Diagnostics.AddError("Error updating dataset", err.Error())
		return
	}

	if err := datasetToModel(ds, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading dataset after update", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatasetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.PreventDeletion.ValueBool() {
		resp.Diagnostics.AddError(
			"Dataset deletion prevented",
			fmt.Sprintf("Dataset %q has `prevent_deletion = true`. Apply with `prevent_deletion = false` before destroying.", state.Path.ValueString()),
		)
		return
	}

	if err := truenas.PoolDatasetDelete(ctx, r.caller, state.Path.ValueString()); err != nil {
		if isNotFoundErr(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting dataset", err.Error())
	}
}

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// ---- conversion helpers ----------------------------------------------------

// defaultPreventDeletion resolves an unset (Null/Unknown) prevent_deletion to
// its false default. prevent_deletion is never returned by the API, so this
// is the only place its default is applied — on Create (no prior state) and
// on Read immediately after import (ImportStatePassthroughID leaves it Null).
// Normal refresh cycles never see Null here since Create/Update already
// resolved it.
func defaultPreventDeletion(m *DatasetResourceModel) {
	if m.PreventDeletion.IsNull() || m.PreventDeletion.IsUnknown() {
		m.PreventDeletion = types.BoolValue(false)
	}
}

// datasetMutableFields holds the 14 mutable ZFS dataset properties in their
// Terraform-model-independent Go form, used to build both create and update
// API args from the same extraction logic.
type datasetMutableFields struct {
	comments      *string
	compression   *string
	sync          *string
	atime         *string
	exec          *string
	readonly      *string
	deduplication *string
	snapDir       *string
	recordSize    *string

	quota          *int64
	refquota       *int64
	reservation    *int64
	refreservation *int64
	copies         *int64
}

func datasetMutableFieldsFromModel(m DatasetResourceModel) datasetMutableFields {
	return datasetMutableFields{
		comments:      stringPtrIfSet(m.Comments),
		compression:   stringPtrIfSet(m.Compression),
		sync:          stringPtrIfSet(m.Sync),
		atime:         stringPtrIfSet(m.Atime),
		exec:          stringPtrIfSet(m.Exec),
		readonly:      stringPtrIfSet(m.Readonly),
		deduplication: stringPtrIfSet(m.Deduplication),
		snapDir:       stringPtrIfSet(m.SnapDir),
		recordSize:    stringPtrIfSet(m.RecordSize),

		quota:          int64PtrIfSet(m.Quota),
		refquota:       int64PtrIfSet(m.Refquota),
		reservation:    int64PtrIfSet(m.Reservation),
		refreservation: int64PtrIfSet(m.Refreservation),
		copies:         int64PtrIfSet(m.Copies),
	}
}

func (f datasetMutableFields) applyToCreateArgs(args *truenas.PoolDatasetCreateArgs) {
	if f.comments != nil {
		args.Comments = *f.comments
	}
	if f.compression != nil {
		args.Compression = *f.compression
	}
	if f.sync != nil {
		args.Sync = *f.sync
	}
	if f.atime != nil {
		args.Atime = *f.atime
	}
	if f.exec != nil {
		args.Exec = *f.exec
	}
	if f.readonly != nil {
		args.Readonly = *f.readonly
	}
	if f.deduplication != nil {
		args.Deduplication = *f.deduplication
	}
	if f.snapDir != nil {
		args.Snapdir = *f.snapDir
	}
	if f.recordSize != nil {
		args.Recordsize = *f.recordSize
	}
	if f.quota != nil {
		args.Quota = int64ToRawMessage(*f.quota)
	}
	if f.refquota != nil {
		args.Refquota = int64ToRawMessage(*f.refquota)
	}
	if f.reservation != nil {
		args.Reservation = *f.reservation
	}
	if f.refreservation != nil {
		args.Refreservation = *f.refreservation
	}
	if f.copies != nil {
		args.Copies = int64ToRawMessage(*f.copies)
	}
}

func (f datasetMutableFields) applyToUpdateArgs(args *truenas.PoolDatasetUpdateArgs) {
	if f.comments != nil {
		args.Comments = *f.comments
	}
	if f.compression != nil {
		args.Compression = *f.compression
	}
	if f.sync != nil {
		args.Sync = *f.sync
	}
	if f.atime != nil {
		args.Atime = *f.atime
	}
	if f.exec != nil {
		args.Exec = *f.exec
	}
	if f.readonly != nil {
		args.Readonly = *f.readonly
	}
	if f.deduplication != nil {
		args.Deduplication = *f.deduplication
	}
	if f.snapDir != nil {
		args.Snapdir = *f.snapDir
	}
	if f.recordSize != nil {
		args.Recordsize = *f.recordSize
	}
	if f.quota != nil {
		args.Quota = int64ToRawMessage(*f.quota)
	}
	if f.refquota != nil {
		args.Refquota = int64ToRawMessage(*f.refquota)
	}
	if f.reservation != nil {
		args.Reservation = *f.reservation
	}
	if f.refreservation != nil {
		args.Refreservation = *f.refreservation
	}
	if f.copies != nil {
		args.Copies = int64ToRawMessage(*f.copies)
	}
}

func stringPtrIfSet(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64PtrIfSet(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := v.ValueInt64()
	return &n
}

func int64ToRawMessage(n int64) json.RawMessage {
	b, _ := json.Marshal(n)
	return b
}

// datasetToModel populates m's identity and mutable-property attributes from
// ds. It never touches prevent_deletion, which the API does not return.
func datasetToModel(ds *truenas.PoolDataset, m *DatasetResourceModel) error {
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

func stringPtrToValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

func int64PtrToValue(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
