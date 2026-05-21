---
page_title: "Provider: MSL5 Live"
description: |-
  The MSL5 Live provider manages Akamai Media Services Live (MSL5) resources
  via the MSL5 REST API, enabling Infrastructure-as-Code workflows for live
  streaming origins, streams, events, and ingest credentials.
---

# MSL5 Live Provider

The **MSL5 Live** provider enables Terraform management of [Akamai Media Services Live (MSL5)](https://www.akamai.com/) resources.

## Example Usage

```terraform
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
```

## Schema

### Required

- `endpoint` (String) Base URL of the MSL5 API.
- `api_token` (String, Sensitive) Bearer token for authentication.

### Optional

- `request_timeout` (Number) HTTP timeout in seconds. Default: `30`.
- `max_retries` (Number) Max retries on 5xx/timeout errors. Default: `3`.
- `debug_logging` (Boolean) Enable debug logging. Default: `false`.
