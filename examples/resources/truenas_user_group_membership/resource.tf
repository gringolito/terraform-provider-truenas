resource "truenas_group" "ops" {
  name = "ops"
}

resource "truenas_group" "dev" {
  name = "dev"
}

resource "truenas_user" "alice" {
  username          = "alice"
  full_name         = "Alice Example"
  group             = truenas_group.ops.id
  password_disabled = true
}

resource "truenas_user_group_membership" "alice_groups" {
  user_id   = truenas_user.alice.id
  group_ids = [truenas_group.ops.id, truenas_group.dev.id]
}
