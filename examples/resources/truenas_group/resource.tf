resource "truenas_group" "example" {
  name = "devteam"
  smb  = true

  sudo_commands = [
    "/usr/bin/systemctl restart myservice",
  ]
}
