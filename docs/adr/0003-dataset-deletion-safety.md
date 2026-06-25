# Dataset deletion safety: provider-level `prevent_deletion` + strict delete

Datasets are the only in-scope resource whose destruction is irreversible (a deleted
dataset is lost data; users/groups/shares are recreatable config). We protect them two
ways:

1. **A provider-level `prevent_deletion` attribute** (bool, default `false`) on the dataset
   resource, enforced inside `Delete`: when `true`, `Delete` returns an error instead of
   destroying. Chosen *in addition to* Terraform's native `lifecycle.prevent_destroy`
   because the native guard must be a literal (no variables), lives only in config (vanishes
   if the resource block is removed), and is easy to forget. The provider flag lives in
   prior state, so it also protects against block removal, `-target`, and — crucially —
   **deletion triggered by a `ForceNew` replacement**. The resulting "stuck" state (must
   `apply` `prevent_deletion = false` before destroying) is intentional friction.

2. **Strict delete**: `pool.dataset.delete` is always called with `recursive=false`,
   `force=false` (the API defaults). The provider never implicitly destroys child datasets,
   snapshots, or busy datasets — such a delete fails and the error is surfaced, forcing the
   user to clean up children explicitly. An opt-in recursive attribute may be added later.

Additionally, `ForceNew` is minimized on the dataset resource (ideally only `name`/path
forces replacement) so accidental replacement — and thus accidental deletion — is rare.

## Consequences

- Scoped to dataset (and future zvol) only; config resources don't carry `prevent_deletion`.
- Docs must loudly explain the flip-and-apply-then-destroy workflow and that the flag also
  blocks replacement.
