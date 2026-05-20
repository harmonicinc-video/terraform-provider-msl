# ---------------------------------------------------------------------------
# Origins
# ---------------------------------------------------------------------------
output "managed_origin_ids" {
  description = "Map of origin key to resource ID."
  value       = { for k, v in msl_origin.managed : k => v.id }
}

output "managed_origin_statuses" {
  description = "Map of origin key to current status."
  value       = { for k, v in msl_origin.managed : k => v.status }
}

output "managed_origin_updated_ats" {
  description = "Map of origin key to RFC3339 timestamp of the last update."
  value       = { for k, v in msl_origin.managed : k => v.updated_at }
}

# ---------------------------------------------------------------------------
# Streams
# ---------------------------------------------------------------------------
output "managed_stream_ids" {
  description = "Map of stream key to resource ID."
  value       = { for k, v in msl_stream.managed : k => v.id }
}

output "managed_stream_primary_publishing_urls" {
  description = "Map of stream key to primary publishing URL."
  value       = { for k, v in msl_stream.managed : k => v.primary_publishing_url }
}

output "managed_stream_statuses" {
  description = "Map of stream key to current status."
  value       = { for k, v in msl_stream.managed : k => v.status }
}

# ---------------------------------------------------------------------------
# Events
# ---------------------------------------------------------------------------
output "managed_event_ids" {
  description = "Map of event key to resource ID."
  value       = { for k, v in msl_event.managed : k => v.id }
}

output "managed_event_names" {
  description = "Map of event key to event name."
  value       = { for k, v in msl_event.managed : k => v.event_name }
}

# ---------------------------------------------------------------------------
# Ingest credentials
# ---------------------------------------------------------------------------
output "managed_ingest_credential_ids" {
  description = "Map of credential key to resource ID."
  sensitive   = true
  value       = { for k, v in msl_ingest_credential.managed : k => v.id }
}

output "managed_ingest_credential_usernames" {
  description = "Map of credential key to username."
  sensitive   = true
  value       = { for k, v in msl_ingest_credential.managed : k => v.username }
}
