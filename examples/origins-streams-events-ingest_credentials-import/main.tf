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
# Origins
# ---------------------------------------------------------------------------
resource "msl_origin" "managed" {
  for_each = var.origins

  host_name              = each.value.host_name
  ingest_location        = each.value.ingest_location
  contract_id            = each.value.contract_id
  cptag                  = each.value.cptag
  group_id               = each.value.group_id
  backup_ingest_location = each.value.backup_ingest_location

  dynamic "shared_keys" {
    for_each = each.value.shared_keys
    content {
      name      = shared_keys.value.name
      key       = shared_keys.value.key
      host_name = shared_keys.value.host_name
      enabled   = shared_keys.value.enabled
    }
  }
}

# ---------------------------------------------------------------------------
# Streams  (reference the imported origin via origin_key)
# ---------------------------------------------------------------------------
resource "msl_stream" "managed" {
  for_each = var.streams

  format                     = each.value.format
  description                = each.value.description
  origin_id                  = msl_origin.managed[each.value.origin_key].id
  ingest_location            = msl_origin.managed[each.value.origin_key].ingest_location
  contract_id                = msl_origin.managed[each.value.origin_key].contract_id
  cptag                      = msl_origin.managed[each.value.origin_key].cptag
  group_id                   = msl_origin.managed[each.value.origin_key].group_id
  ingest_authentication_mode = each.value.ingest_authentication_mode
  allowed_ips                = each.value.allowed_ips
  playlist_duration_in_min   = each.value.playlist_duration_in_min

  dynamic "hls_to_llhls" {
    for_each = each.value.hls_to_llhls != null ? [each.value.hls_to_llhls] : []
    content {
      enabled          = hls_to_llhls.value.enabled
      segment_template = hls_to_llhls.value.segment_template
    }
  }

  dynamic "ingest_header" {
    for_each = each.value.ingest_header != null ? [each.value.ingest_header] : []
    content {
      header = ingest_header.value.header
      values = ingest_header.value.values
    }
  }

  dynamic "playback" {
    for_each = each.value.playback != null ? [each.value.playback] : []
    content {
      dynamic "akamai_g2o_auth" {
        for_each = playback.value.akamai_g2o_auth != null ? [playback.value.akamai_g2o_auth] : []
        content {
          enabled     = akamai_g2o_auth.value.enabled
          g2o_version = akamai_g2o_auth.value.g2o_version
          secret_key  = akamai_g2o_auth.value.secret_key
          time_delta  = akamai_g2o_auth.value.time_delta
        }
      }
    }
  }

  dynamic "archiving" {
    for_each = each.value.archiving != null ? [each.value.archiving] : []
    content {
      no_archive = archiving.value.no_archive
      dynamic "automatic_purge" {
        for_each = archiving.value.automatic_purge != null ? [archiving.value.automatic_purge] : []
        content {
          retention_days = automatic_purge.value.retention_days
        }
      }
    }
  }
}

# ---------------------------------------------------------------------------
# Events  (reference the imported stream via stream_key)
# ---------------------------------------------------------------------------
resource "msl_event" "managed" {
  for_each = var.events

  stream_id           = msl_stream.managed[each.value.stream_key].id
  event_name          = each.value.event_name
  source_event_name   = each.value.source_event_name
  start_time          = each.value.start_time
  end_time            = each.value.end_time
  subsegment_clipping = each.value.subsegment_clipping
}

# ---------------------------------------------------------------------------
# Ingest credentials  (reference the imported stream via stream_key)
# ---------------------------------------------------------------------------
resource "msl_ingest_credential" "managed" {
  for_each = var.ingest_credentials

  stream_id   = msl_stream.managed[each.value.stream_key].id
  username    = each.value.username
  password    = each.value.password
  algorithm   = each.value.algorithm
  description = each.value.description
}
