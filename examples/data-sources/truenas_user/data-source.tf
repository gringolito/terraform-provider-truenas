# Look up by Unix UID
data "truenas_user" "by_uid" {
  id = 1001
}

# Look up by username
data "truenas_user" "by_username" {
  username = "jdoe"
}
