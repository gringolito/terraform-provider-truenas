# Generate the typed client from a checked-in method-registry snapshot

`truenas/api_client_golang` is an untyped JSON-RPC client (`Call(method, params) →
json.RawMessage`); it ships no Go models for users, datasets, or shares. A spike against a
live TrueNAS SCALE 25.10.4 box confirmed `core.get_methods` returns rich, fully-inlined
TrueNAS-flavoured JSON-Schema for all 769 methods (typed properties, `anyOf` nullables,
enums, per-property `_required_`, `_attrs_order_`, polymorphic `anyOf`/`oneOf` payloads,
**no `$ref`**). We therefore generate the **typed client layer only** from a checked-in
registry snapshot, and **hand-write the Terraform resource layer** on top of it.

Generating the typed client gives leverage where the API is regular; hand-writing
resources keeps the editorial Terraform decisions (Required/Optional/Computed, Sensitive,
ForceNew, defaults, import identity, drift normalization) under human control — these are
not mechanically derivable and codegen of that layer reliably produces resources that
compile but misbehave.

## Consequences

- The generator must handle: multi-positional-arg methods (`update` = `[id, {patch}]`),
  JSON-Schema composition (`anyOf`/`oneOf`/`allOf`), pervasive `anyOf:[T, null]`
  nullables, and TrueNAS's `_required_`/`_attrs_order_` extensions. It does **not** need
  `$ref` resolution.
- The snapshot is pinned to a TrueNAS version; re-snapshotting is a deliberate, reviewed
  step so generation is reproducible in CI without a live server.
