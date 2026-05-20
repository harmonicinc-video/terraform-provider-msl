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

variable "request_timeout" {
  type        = number
  default     = 30
  description = "HTTP request timeout in seconds."
}

variable "max_retries" {
  type        = number
  default     = 3
  description = "Maximum number of retries on transient errors."
}

variable "debug_logging" {
  type        = bool
  default     = false
  description = "Enable verbose debug logging (never logs sensitive values)."
}

# ---------------------------------------------------------------------------
# Data source filters
# ---------------------------------------------------------------------------
variable "contract_id" {
  type        = string
  default     = null
  description = "Optional contract ID to filter origins and streams. Omit to list all."
}

variable "group_id" {
  type        = string
  default     = null
  description = "Optional group ID to filter origins and streams. Omit to list all."
}
