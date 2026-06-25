# Project conventions

## Git conventions

All commits must be signed and signed-off. Use `git commit -S -s`.

## PR checklist

Before opening any PR that is not documentation-only, run:

1. `make fmt` - apply code format rules
2. `make generate` — regenerate any auto-generated files
3. `make build` — ensure the project compiles
4. `make lint` — ensure there are no lint errors
5. `make acc-tests` — run both unit tests and acceptance tests

## Docs generation

`make generate` uses `tfplugindocs` to auto-generate `docs/resources/*.md` and
`docs/data-sources/*.md` from two sources:

- **Schema descriptions** — the `MarkdownDescription` fields on each resource/data source schema attribute.
- **Examples** — `.tf` files under `examples/resources/<resource_name>/resource.tf` and
  `examples/data-sources/<data_source_name>/data-source.tf`.

When adding or changing a resource or data source, always update the corresponding example file
under `examples/` before running `make generate`. Never edit the generated files under `docs/`
directly — they will be overwritten.

## Agent skills

### Issue tracker

Issues and PRDs live in **GitHub Issues** (`gringolito/terraform-provider-truenas`), tracked
in GitHub Project #5; use the `gh` CLI. External PRs are **not** a triage surface. See
`docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles. `needs-info` maps to the existing `needs-clarification` label
and `wontfix` reuses the existing label; the rest use default names. See
`docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
