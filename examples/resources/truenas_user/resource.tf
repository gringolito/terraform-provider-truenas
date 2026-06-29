resource "truenas_group" "mygroup" {
  name = "mygroup"
}

resource "truenas_user" "example" {
  username  = "jdoe"
  full_name = "Jane Doe"
  group     = truenas_group.mygroup.id

  # Password is write-only and never stored in state.
  # Bump password_wo_version to trigger a password change.
  password            = "s3cr3t!"
  password_wo_version = 1

  smb  = true
  home = "/home/jdoe"
}
