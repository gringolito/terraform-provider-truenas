resource "truenas_dataset" "example" {
  path        = "tank/projects/myapp"
  comments    = "Storage for myapp"
  compression = "LZ4"
  atime       = "OFF"
  quota       = 107374182400 # 100 GiB

  # Refuse to destroy this dataset, even via `terraform destroy` or a
  # `path`-triggered replacement. Flip to `false` and apply before removing it.
  prevent_deletion = true
}
