# Look up a group by its integer API ID
data "truenas_group" "by_id" {
  id = 1001
}

# Look up a group by name
data "truenas_group" "by_name" {
  name = "admins"
}
