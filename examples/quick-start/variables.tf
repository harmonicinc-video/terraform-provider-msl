# ---------------------------------------------------------------------------
# Provider configuration
# ---------------------------------------------------------------------------
variable "api_endpoint" {
  type        = string
  description = "Base URL of the MSL5 API, e.g. https://api.msl.example.com"
}

variable "api_token" {
  type        = string
  sensitive   = true
  description = "Bearer token for MSL5 API authentication."
}
