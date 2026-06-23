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
  ephemeral   = true
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
# Origins
# ---------------------------------------------------------------------------
variable "origins" {
  description = "Map of origin key to origin configuration."
  type = map(object({
    host_name              = string
    ingest_location        = string
    contract_id            = string
    cptag                  = string
    group_id               = string
    backup_ingest_location = optional(string)
    shared_keys = optional(list(object({
      name      = string
      key       = string
      host_name = string
      enabled   = bool
    })), [])
  }))
  default = {}
}

# ---------------------------------------------------------------------------
# Streams
# ---------------------------------------------------------------------------
variable "streams" {
  description = "Map of stream key to stream configuration. origin_key must match a key in var.origins."
  type = map(object({
    origin_key                 = string
    format                     = string
    description                = optional(string, "")
    ingest_authentication_mode = optional(string, "DIGEST")
    allowed_ips                = optional(list(string), ["0.0.0.0/0"])
    playlist_duration_in_min   = optional(number)
    hls_to_llhls = optional(object({
      enabled          = bool
      segment_template = optional(string)
    }))
    ingest_header = optional(object({
      header = string
      values = list(string)
    }))
    playback = optional(object({
      akamai_g2o_auth = optional(object({
        enabled     = optional(bool)
        g2o_version = optional(number)
        secret_key  = optional(string)
        time_delta  = optional(number)
      }))
    }))
    archiving = optional(object({
      no_archive = bool
      automatic_purge = optional(object({
        retention_days = number
      }))
    }))
  }))
  default = {}
}

# ---------------------------------------------------------------------------
# Events
# ---------------------------------------------------------------------------
variable "events" {
  description = "Map of event key to event configuration. stream_key must match a key in var.streams."
  type = map(object({
    stream_key          = string
    event_name          = string
    source_event_name   = optional(string)
    start_time          = optional(string)
    end_time            = optional(string)
    subsegment_clipping = optional(bool)
  }))
  default = {}
}

# ---------------------------------------------------------------------------
# Ingest credentials
# ---------------------------------------------------------------------------
variable "ingest_credentials" {
  description = "Map of credential key to ingest credential configuration. stream_key must match a key in var.streams."
  type = map(object({
    stream_key  = string
    username    = string
    password    = string
    algorithm   = optional(string, "SHA256")
    description = optional(string, "")
  }))
  default = {}
}
