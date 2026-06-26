# Additive-only, asymmetrical user–group membership

Supplementary user↔group membership is bidirectional in the TrueNAS API — settable via both
`user.update(groups=…)` and `group.update(users=…)`. If `truenas_user` managed `groups` and
`truenas_group` managed `users` inline, two resources would fight over one relationship and
produce perpetual drift. We adopt the AWS owner/consumer/attachment pattern: the `truenas_user`
and `truenas_group` resources do **not** manage supplementary membership inline (`user.groups`
and `group.users` are Computed/read-only); a dedicated attachment resource owns it.

For v0.1 we ship **only an additive attachment**, `truenas_user_group_membership` (user-anchored;
manages just the `(user, group)` edges it declares, via read-modify-write of `user.groups`) —
mirroring `aws_iam_user_group_membership`. Multiple attachments compose as a union with no
flapping. We deliberately do **not** ship an exclusive variant now; the name
`truenas_group_membership` is reserved for a future *exclusive* resource (mirroring
`aws_iam_group_membership`) if authoritative rosters are ever needed.

A user's single **primary group** (`user.group`) is intrinsic identity, not membership, and
remains a required attribute on `truenas_user`.

## Considered and rejected

- **Exclusive-only.** Gives authoritative rosters and out-of-band claw-back, but disallows
  composition (one membership resource per group). Composition was valued more for v0.1.
- **Ship both additive and exclusive with a collision-detecting marker.** Rejected as
  infeasible: there is nowhere to put the marker and no way to detect the collision. TrueNAS
  groups expose no free-form metadata field (only `gid`/`name`/`sudo_commands`/
  `sudo_commands_nopasswd`/`smb`/`userns_idmap`/`users`), so an ownership marker could only be
  stored by corrupting a real field. And Terraform gives a resource no cross-resource visibility
  at plan time and no cross-state visibility ever — so a guard would catch only same-config
  collisions and silently miss cross-state ones. Mixing exclusive + additive on one group
  oscillates forever (e.g. flapping between `{X,Y,Z}` and `{X,Y,Z,W}`); the reliable fix is to
  make the collision unrepresentable by shipping one model, which is why AWS documents the
  incompatibility rather than detecting it.

## Consequences

- No authoritative-roster guarantee: Terraform will not claw back a member added out-of-band.
  Acceptable for v0.1; recoverable later by adding the reserved exclusive variant.
- `truenas_user.groups` and `truenas_group.users` are Computed reflections, never managed inline.
