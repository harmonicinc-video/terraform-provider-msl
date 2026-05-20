terraform {
  required_version = ">= 1.14"

  required_providers {
    msl = {
      source  = "harmonicinc-video/msl"
      version = "~> 0.1"
    }
  }
}

provider "msl" {
  endpoint  = var.api_endpoint
  api_token = var.api_token

  request_timeout = var.request_timeout
  max_retries     = var.max_retries
  debug_logging   = var.debug_logging
}

# ---------------------------------------------------------------------------
# Data Source: List MSL5 origins, optionally filtered by contract and group.
# ---------------------------------------------------------------------------
data "msl_origins" "all" {
  contract_id = var.contract_id
  group_id    = var.group_id
}

# ---------------------------------------------------------------------------
# Data Source: List MSL5 streams, optionally filtered by contract and group.
# ---------------------------------------------------------------------------
data "msl_streams" "all" {
  contract_id = var.contract_id
  group_id    = var.group_id
}

# ---------------------------------------------------------------------------
# Data Source: List MSL5 events for each stream returned above.
# ---------------------------------------------------------------------------
data "msl_events" "by_stream" {
  for_each  = { for s in data.msl_streams.all.streams : s.id => s }
  stream_id = each.key
}

# ---------------------------------------------------------------------------
# Data Source: List MSL5 ingest credentials for each stream returned above.
# ---------------------------------------------------------------------------
data "msl_ingest_credentials" "by_stream" {
  for_each  = { for s in data.msl_streams.all.streams : s.id => s }
  stream_id = each.key
}
