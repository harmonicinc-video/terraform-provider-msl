# ---------------------------------------------------------------------------
# Origin
# ---------------------------------------------------------------------------
output "origin" {
  description = "Managed MSL5 origin attributes."
  value = {
    id                     = msl_origin.terraformtest.id
    host_name              = msl_origin.terraformtest.host_name
    ingest_location        = msl_origin.terraformtest.ingest_location
    backup_ingest_location = msl_origin.terraformtest.backup_ingest_location
    contract_id            = msl_origin.terraformtest.contract_id
    cptag                  = msl_origin.terraformtest.cptag
    group_id               = msl_origin.terraformtest.group_id
    shared_keys            = msl_origin.terraformtest.shared_keys
    status                 = msl_origin.terraformtest.status
    created_by             = msl_origin.terraformtest.created_by
    created_at             = msl_origin.terraformtest.created_at
    updated_at             = msl_origin.terraformtest.updated_at
  }
}

# ---------------------------------------------------------------------------
# Stream
# ---------------------------------------------------------------------------
output "stream" {
  description = "Managed MSL5 stream attributes."
  value = {
    id                         = msl_stream.terraformtest.id
    format                     = msl_stream.terraformtest.format
    description                = msl_stream.terraformtest.description
    host_name                  = msl_stream.terraformtest.host_name
    primary_publishing_url     = msl_stream.terraformtest.primary_publishing_url
    backup_host_name           = msl_stream.terraformtest.backup_host_name
    backup_publishing_url      = msl_stream.terraformtest.backup_publishing_url
    ingest_location            = msl_stream.terraformtest.ingest_location
    backup_ingest_location     = msl_stream.terraformtest.backup_ingest_location
    origin_id                  = msl_stream.terraformtest.origin_id
    origin_host_name           = msl_stream.terraformtest.origin_host_name
    backup_origin_host_name    = msl_stream.terraformtest.backup_origin_host_name
    contract_id                = msl_stream.terraformtest.contract_id
    cptag                      = msl_stream.terraformtest.cptag
    group_id                   = msl_stream.terraformtest.group_id
    allowed_ips                = msl_stream.terraformtest.allowed_ips
    archiving                  = msl_stream.terraformtest.archiving
    hls_to_llhls               = msl_stream.terraformtest.hls_to_llhls
    ingest_authentication_mode = msl_stream.terraformtest.ingest_authentication_mode
    ingest_header              = msl_stream.terraformtest.ingest_header
    playback                   = msl_stream.terraformtest.playback
    playlist_duration_in_min   = msl_stream.terraformtest.playlist_duration_in_min
    status                     = msl_stream.terraformtest.status
    created_by                 = msl_stream.terraformtest.created_by
    created_at                 = msl_stream.terraformtest.created_at
    updated_at                 = msl_stream.terraformtest.updated_at
  }
}

# ---------------------------------------------------------------------------
# Event
# ---------------------------------------------------------------------------
output "event" {
  description = "Managed MSL5 event attributes."
  value = {
    id                  = msl_event.terraformtest.id
    stream_id           = msl_event.terraformtest.stream_id
    event_name          = msl_event.terraformtest.event_name
    source_event_name   = msl_event.terraformtest.source_event_name
    start_time          = msl_event.terraformtest.start_time
    end_time            = msl_event.terraformtest.end_time
    active              = msl_event.terraformtest.active
    ended               = msl_event.terraformtest.ended
    egress_filepaths    = msl_event.terraformtest.egress_filepaths
    subsegment_clipping = msl_event.terraformtest.subsegment_clipping
    created_at          = msl_event.terraformtest.created_at
    updated_at          = msl_event.terraformtest.updated_at
  }
}

# ---------------------------------------------------------------------------
# Ingest credential
# ---------------------------------------------------------------------------
output "ingest_credential" {
  description = "Managed MSL5 ingest credential attributes."
  sensitive   = true
  value = {
    id          = msl_ingest_credential.managed.id
    stream_id   = msl_ingest_credential.managed.stream_id
    username    = msl_ingest_credential.managed.username
    password    = msl_ingest_credential.managed.password
    algorithm   = msl_ingest_credential.managed.algorithm
    description = msl_ingest_credential.managed.description
    expiry_date = msl_ingest_credential.managed.expiry_date
  }
}
