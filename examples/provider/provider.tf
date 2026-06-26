# API key authentication (recommended)
provider "truenas" {
  host    = "truenas.example.com"
  api_key = "your-api-key"
}

# Username and password authentication
provider "truenas" {
  host     = "truenas.example.com"
  username = "admin"
  password = "your-password"
}

# Using environment variables (TRUENAS_HOST, TRUENAS_API_KEY, etc.)
provider "truenas" {}
