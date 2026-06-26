# Write-only secrets; minimum Terraform 1.11

TrueNAS never returns secrets (`user.get_instance` omits `password`), so a secret can never
be refreshed into state and gains nothing from being stored there. We therefore model
secret inputs as **write-only arguments** (e.g. `password` / `password_wo_version`): the
value is supplied at apply time and **never persisted to Terraform state**, with a companion
`_version` attribute to trigger updates. This is the default convention for *all* secret
inputs in this provider (future iSCSI/SMB auth secrets follow suit).

Write-only arguments require **Terraform ≥ 1.11**, which becomes the provider's minimum
supported version. Chosen over storing a `Sensitive` value in state because keeping
plaintext secrets out of the state file is the stronger default for a new provider, and a
2026 TF-version floor is uncontroversial.

## Consequences

- Secrets are absent from state; a secret changed out-of-band on TrueNAS is not drift-detected
  (unavoidable — it can't be read back). Rotation is driven by bumping the `_version`.
- The provider declares a Terraform 1.11 floor; older Terraform is unsupported.
