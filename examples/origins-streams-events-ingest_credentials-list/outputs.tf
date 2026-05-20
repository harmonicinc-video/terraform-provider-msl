# ---------------------------------------------------------------------------
# Origins
# ---------------------------------------------------------------------------
output "origins" {
  description = "All MSL5 origins returned by the data source."
  value       = data.msl_origins.all.origins
}

output "origin_ids" {
  description = "List of origin IDs."
  value       = [for o in data.msl_origins.all.origins : o.id]
}

output "origin_statuses" {
  description = "Map of origin ID to current status."
  value       = { for o in data.msl_origins.all.origins : o.id => o.status }
}

# ---------------------------------------------------------------------------
# Streams
# ---------------------------------------------------------------------------
output "streams" {
  description = "All MSL5 streams returned by the data source."
  value       = data.msl_streams.all.streams
}

output "stream_ids" {
  description = "List of stream IDs."
  value       = [for s in data.msl_streams.all.streams : s.id]
}

output "stream_primary_publishing_urls" {
  description = "Map of stream ID to primary publishing URL."
  value       = { for s in data.msl_streams.all.streams : s.id => s.primary_publishing_url }
}

output "stream_statuses" {
  description = "Map of stream ID to current status."
  value       = { for s in data.msl_streams.all.streams : s.id => s.status }
}

# ---------------------------------------------------------------------------
# Events (grouped by stream)
# ---------------------------------------------------------------------------
output "events_by_stream" {
  description = "Map of stream ID to the list of events for that stream."
  value       = { for stream_id, ds in data.msl_events.by_stream : stream_id => ds.events }
}
