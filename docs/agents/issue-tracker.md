# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub Issues, managed via the `github-backlog-management-skill`. Use the `gh` CLI for all operations.

## Conventions

- **Read an issue**: `gh issue view <number> --comments`, filtering comments by `jq` and also fetching labels.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with appropriate `--label` and `--state` filters.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v` — `gh` does this automatically when run inside a clone.

## When a skill says "publish to the issue tracker"

Call the `/add-backlog-item` skill — do not call `gh issue create` directly.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.
