# User-facing identifiers use Unix UIDs / GIDs, not internal API IDs

TrueNAS has two ID spaces for every identity primitive:

- **Internal API ID** (`id` on the JSON object) — an opaque integer used in
  `user.create(group=<id>)`, `group.update(users=[<id>, ...])`, etc. Not stable across
  backup/restore and meaningless to operators.
- **Unix UID / GID** — the POSIX identifier the OS and share layer actually use. Stable,
  visible in `ls -l` / `/etc/passwd` output, meaningful to operators.

Every Terraform attribute that identifies or references a user or group exposes the **Unix UID
or GID**, never the internal API ID. This includes the resource's own `id` attribute and
consequently the import ID (e.g. `terraform import truenas_user.foo 1001`).

| Attribute | Exposes |
|---|---|
| `truenas_user.id` | Unix UID of the user |
| `truenas_user.group` (primary group, required) | Unix GID of the primary group |
| `truenas_user.groups` (supplementary, computed/read-only) | Unix GIDs of supplementary groups |
| `truenas_group.id` | Unix GID of the group |
| `truenas_group.users` (computed/read-only) | Unix UIDs of member users |

## Internal ID handling

The internal API ID is required for `user.update`, `user.delete`, `group.update`, and
`group.delete`, which all take it as the first positional argument. Since it is not stored in
any user-visible attribute, it is persisted via the plugin framework's
`resource.ResourceWithPrivateState` interface (opaque bytes stored alongside state but never
surfaced in `terraform show` or the state file's human-readable output).

On every `Read`, the provider fetches the resource by UID/GID (via `user.query`/`group.query`)
and refreshes the private internal ID. If the internal ID changes across a backup/restore
cycle, the next `Read` transparently re-anchors it.

## Resolution calls

| Operation | Extra call | Reason |
|---|---|---|
| `truenas_user` Create / Update | `group.query(gid=N)` | Resolve primary-group GID → internal API ID |
| `truenas_user` Read | `group.query(id in [...])` | Resolve supplementary group IDs → GIDs |
| `truenas_group` Read | `user.query(id in [...])` | Resolve member user IDs → UIDs (already implemented) |
| `truenas_user` Import | `user.query(uid=N)` | Seed initial state + private internal ID |
| `truenas_group` Import | `group.query(gid=N)` | Seed initial state + private internal ID |

## Consequences

- **`truenas_group.id` is a breaking change** relative to the initial implementation, which
  used the internal API ID. A follow-up issue should land a `truenas_group` fix (change `id`
  to GID, switch to private state for the internal ID, update import, update tests). Until
  that fix lands, `truenas_group` is inconsistent with this ADR; `truenas_user` is built
  correctly from the start.
- **Natural cross-resource references**: `group = truenas_group.mygroup.gid` is no longer
  needed — with `truenas_group.id` = GID, operators write `group = truenas_group.mygroup.id`,
  the same pattern used everywhere else in Terraform.
- **No internal IDs visible to operators**: `terraform show`, state files, and plan output
  never contain an opaque internal ID. Operators only ever see UIDs and GIDs.

## Why not store an internal `api_id` attribute in the visible schema?

A visible `api_id` attribute would appear in `terraform show`, `terraform state show`, and
plan output, and operators would be confused by it. Private state is the correct tool for
bookkeeping values the provider needs but operators should never see or configure.
