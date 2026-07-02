# terraform-provider-truenas

A Terraform provider for TrueNAS SCALE, built on the official `truenas/api_client_golang`
WebSocket JSON-RPC client. This glossary fixes the vocabulary the codebase and design
discussions use.

## Language

**Middleware method**:
A remote procedure exposed by the TrueNAS middleware over the JSON-RPC API, named
`namespace.verb` (e.g. `user.create`, `pool.dataset.update`). The unit of interaction
with TrueNAS.
_Avoid_: endpoint, API call, RPC (when you mean the named operation)

**Sync method**:
A middleware method that returns its result directly in the response.

**Job method**:
A middleware method that returns a numeric job id and runs asynchronously; the caller
tracks completion via job updates. Distinct from a sync method because the result is not
in the immediate response.
_Avoid_: async method, long-running call

**Envelope**:
The JSON-RPC 2.0 wrapper around every response (`{jsonrpc, id, result | error}`). The
client returns this raw for sync calls — it does not unwrap `result` or raise `error` as
a Go error.

**Method registry**:
The full set of middleware method definitions returned by `core.get_methods`, each
carrying a TrueNAS-flavoured JSON-Schema for its arguments (`accepts`) and result
(`returns`).

**Registry snapshot**:
A checked-in JSON dump of the method registry from a pinned TrueNAS version. The input
the code generator consumes; re-snapshotting is a deliberate, reviewed step.

**Typed client**:
The generated Go layer (`internal/...`) that wraps the raw `api_client_golang` client and
exposes typed methods/structs derived from the registry snapshot. The provider's resource
layer is hand-written on top of it and never touches raw JSON.
_Avoid_: SDK, wrapper (be specific)

**Prevent-deletion**:
A provider-level safety attribute on the dataset resource that makes `Delete` refuse to
destroy when set, guarding irreversible data loss (including from `ForceNew` replacement).
Distinct from Terraform's native `lifecycle.prevent_destroy`.

**Strict delete**:
The provider's policy of never passing `recursive`/`force` to `pool.dataset.delete` — a
dataset with children, snapshots, or active consumers fails to delete rather than cascading.

**Primary group**:
The single, mandatory group intrinsic to a user (`user.group`). Identity, not membership —
it is a required attribute of the `truenas_user` resource, distinct from [[membership]].

**Membership**:
The many-to-many *supplementary* association between users and groups (the TrueNAS
`user.groups` / `group.users` relationship). Managed only through the additive
[[membership-attachment]]; never set inline on the user or group resources.
_Avoid_: groups (when you mean the relationship), supplementary groups (in code identifiers)

**Membership-attachment**:
The dedicated, additive resource (`truenas_user_group_membership`) that owns supplementary
[[membership]]. It manages only the user↔group edges it declares; multiple attachments
compose as a union. `truenas_user.groups` and `truenas_group.users` are read-only
reflections of the resulting state, never authoritative inputs.

**Credential set**:
One of the two valid ways to authenticate with TrueNAS: either an `api_key` alone, or a
`username` + `password` pair together. Provider configuration requires exactly one credential
set — supplying neither, both, or a partial pair (username without password or vice versa) is
a configuration error. Env-var fallbacks (`TRUENAS_API_KEY`, `TRUENAS_USERNAME`,
`TRUENAS_PASSWORD`) are merged before this validation runs.
_Avoid_: credentials (when you mean the full validation constraint), auth method

**Permission / ACL**:
The ownership, POSIX mode, and access-control entries that govern access to a dataset's
files. These live entirely in the TrueNAS `filesystem.*` namespace
(`filesystem.setperm`/`chown`/`setacl`), are **out of v0.1 scope**, and are *not* set by
`pool.dataset.create`. In v0.1, users/groups/[[membership]] are identity primitives only and
grant no dataset or share access by themselves.

**Dataset path**:
The full ZFS path (e.g. `tank/projects/child`) that is a dataset's sole identity. There is
no separate "pool" input on `pool.dataset.create` — the API's `name` field *is* the full
path, and `pool.dataset.get_instance`/`.update`/`.delete` key on this same string. `pool`
(the leading path segment) and `name` (the last segment) exist only as derived, read-only
views split from the path — never as independent create/import inputs.
_Avoid_: dataset name (ambiguous with the API's overloaded `name` field — say "path" for the
full identity, "last path segment" if you specifically mean the tail component)

**Dataset type**:
The discriminator (`FILESYSTEM` or `VOLUME`) on `pool.dataset.create`'s two-variant accepts
schema, selecting which property set applies (e.g. `VOLUME` requires `volsize`). v0.1 manages
`FILESYSTEM` only; encountering `VOLUME` on read/import is a clear, actionable error, not
silently ignored.

**ZFS property**:
A dataset attribute (e.g. `compression`, `quota`) returned by `pool.dataset.get_instance` as
an object with `{value, rawvalue, parsed, source, source_info}`, not a plain scalar.
`source` indicates where the effective value comes from (`LOCAL`, `INHERITED`, `DEFAULT`).
The four representations disagree in format — `value` mirrors the exact string/enum casing
`pool.dataset.create`/`.update` accept (e.g. `"LZ4"`, `"128K"`), while `parsed` is a
Python-native convenience type that does *not* always match the create/update input shape
(e.g. `quota`'s `value` is human-formatted `"20 GiB"` while its `accepts` type is a raw
integer byte count — `parsed` matches there instead). Which of the two to use for a given
property is therefore driven by that property's `accepts` type, not guessable from the read
shape alone.
_Avoid_: raw value (ambiguous between `rawvalue` and `value`)
