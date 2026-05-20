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

resource "msl_origin" "terraformtest" {
  host_name       = "uswestterraformtest"
  ingest_location = "US_SEA"
  contract_id     = "0"
  cptag           = "0"
  group_id        = "dummy"
}

resource "msl_stream" "terraformtest" {
  format                     = "HLS"
  description                = "uswestterraformtest stream"
  origin_id                  = msl_origin.terraformtest.id
  ingest_location            = msl_origin.terraformtest.ingest_location
  contract_id                = msl_origin.terraformtest.contract_id
  cptag                      = msl_origin.terraformtest.cptag
  group_id                   = msl_origin.terraformtest.group_id
  ingest_authentication_mode = "DIGEST"
  archiving {
    no_archive = false
    automatic_purge {
      retention_days = 30
    }
  }
  allowed_ips = ["0.0.0.0/0"]
}

resource "msl_event" "terraformtest" {
  stream_id  = msl_stream.terraformtest.id
  event_name = "terraformtest_event"
}

resource "msl_ingest_credential" "managed" {
  stream_id   = msl_stream.terraformtest.id
  username    = "encoder_live"
  password    = "Ch@ngeme123!"
  algorithm   = "SHA256"
  description = "Active encoding crediential for ${msl_stream.terraformtest.id}"
}
