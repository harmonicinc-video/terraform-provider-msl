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
  endpoint  = var.api_endpoint # e.g. "gateway.mslapis.net"
  api_token = var.api_token    # MSL5 API Bearer token (sensitive)
}

resource "msl_origin" "terraformtest" {
  host_name       = "uswestterraformtest"
  ingest_location = "US_SEA"
  contract_id     = "0"
  cptag           = "0"

  group_id = "dummy"
}

output "origin_id" { value = msl_origin.terraformtest.id }
