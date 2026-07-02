# Dataset codegen extensions: string IDs, discriminated-union accepts, shared ZFS-property type

`pool.dataset.*` needed three generator capabilities `truenas_group`/`truenas_user` never
exercised, all discovered while building the tracer bullet for `truenas_dataset` (#38).

## 1. String-typed first accepts arg as an id-like positional param

`classifyMethod` only recognized an **integer** first `accepts` entry as an id (the
`user`/`group` pattern). Dataset identity is the ZFS path string (`pool.dataset.get_instance`,
`.update`, `.delete` all key on it), so the generator now accepts either an integer or a
string in that position. Not a real trade-off — datasets simply have string ids, and nothing
about the existing behavior needed to change to support them.

## 2. Discriminated-union `accepts` → generation-config branch selection

`pool.dataset.create`'s args are a two-variant `anyOf` discriminated by a `type` const
(`FILESYSTEM` requires `name`; `VOLUME` requires `name`+`volsize`). We taught the generator a
general rule — when an `accepts` entry is an `anyOf` of objects each carrying a `const`
discriminator, generation config selects one branch by that value (e.g.
`pool.dataset.create:type=FILESYSTEM`) — rather than hardcoding this one call site.

**Considered and rejected:** hardcoding the FILESYSTEM selection directly in `classifyMethod`
for `pool.dataset.create` specifically. Rejected even though this pattern has exactly one
call site in all of v0.1 (`sharing.nfs`/`sharing.smb` create/update are plain objects) —
we chose to pay the extra config-surface cost now so a second discriminated-union method
later (e.g. a future zvol/`VOLUME` resource sharing this same RPC) doesn't need the
generator revisited.

## 3. Shared ZFS-property struct + per-property value/parsed extraction table

Every ZFS-backed dataset property (`compression`, `quota`, `atime`, ...) repeats the same
inline `{value, rawvalue, parsed, source, source_info}` shape with no `$ref` (ADR-0001).
The generator now recognizes this recurring shape and emits one shared struct instead of a
one-off per property.

That struct alone doesn't tell the resource layer which field to read: live probing
(`Man/FileShare`) showed `value` and `parsed` disagree in format per property, and neither
is uniformly correct. `value` matches the exact enum/string casing `accepts` requires (e.g.
`compression`: `value="LZ4"` matches, `parsed="lz4"` doesn't); `parsed` matches for
integer-`accepts` properties where `value` is human-formatted and unparseable back into the
input shape (e.g. `quota`: `accepts` wants a raw byte integer, but `value="20 GiB"` while
`parsed=21474836480`). The hand-written `pool_dataset_client.go` therefore keys extraction
off each property's `accepts` type: string/enum → `value`; integer → `parsed`. Getting this
wrong silently breaks "empty plan after apply" for that property.

**Considered and rejected:** leaving the per-property struct un-deduplicated (plain
`map[string]json.RawMessage` per field, as originally generated) and doing all shape-parsing
by hand in `pool_dataset_client.go`. Rejected for the same reason as #2 — one shared,
generator-recognized type now, on the bet that more ZFS-property-shaped fields arrive with
future dataset/zvol work.

## Consequences

- `classifyMethod`/generation config now carries two new capabilities (id-type widening,
  discriminator selection) that only `pool.dataset` exercises today; they are unexercised
  generality until a second consumer appears.
- The value-vs-parsed table lives in hand-written code, not the generator — it depends on
  cross-referencing `accepts` (create/update) against `returns` (get_instance), which the
  generator does not correlate across methods.
