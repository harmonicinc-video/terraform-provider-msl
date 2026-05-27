# Terraform Provider for Media Services Live 5 (MSL5)

A [Terraform](https://www.terraform.io/) provider for [Media Services Live 5 (MSL5)](https://techdocs.akamai.com/msl5-harmonic/docs/welcome-to-msl5) that enables Infrastructure-as-Code management of MSL5 streaming resources.

This provider has NOT been registered to the [HashiCorp Terraform Registry](https://registry.terraform.io/). Before then, you can build and use it locally by referring to Quick Start instructions. Terraform CLI must be run with `TF_CLI_CONFIG_FILE` environment variable pointing to the provided `dev.terraformrc` file that configures a dev override to load the provider plugin from the local directory.
We will further update the instruction when the provider is registered.

## Status

| Resource | CRUD | Import | Data Source |
|---|---|---|---|
| `msl_origin` | ✅ | ✅ | ✅ (`msl_origins`) |
| `msl_stream` | ✅ | ✅ | ✅ (`msl_streams`) |
| `msl_event` | ✅ (no update) | ✅ | ✅ (`msl_events`) |
| `msl_ingest_credential` | ✅ | ✅ | ✅ (`msl_ingest_credentials`) |

## Pre-requisites

- [Terraform](https://www.terraform.io/downloads)
  - This provider has been tested against Terraform 1.14, but newer versions may also work fine
- [Go](https://go.dev/dl/)
  - Required if building the provider plugin from source but not if using pre-built binaries.
  - This provider has been built / tested against Go 1.25.
- MSL5 API endpoint and Bearer token
  - Please obtain these according to [Get started with MSL5 API](https://techdocs.akamai.com/msl5-harmonic/reference/get-started)

## Quick Start (Using Pre-built Binaries)

### 1. Download the latest release

Download the appropriate binary for your platform from the [Releases](https://github.com/harmonicinc-video/terraform-provider-msl/releases) page.

For example, download the compressed file (e.g. `terraform-provider-msl_0.1.0_darwin_arm64.tar.gz`) to designated directory (e.g. /tmp/terraform-provider-msl) for Darwin ARM64. Unzip and verify the checksum against the Release page, then you can use the extracted binary (e.g. `terraform-provider-msl_v0.1.0`) in the next steps.

```bash
$ cd /tmp/terraform-provider-msl
$ shasum -a 256 terraform-provider-msl_0.1.0_darwin_arm64.tar.gz
<checksum> terraform-provider-msl_0.1.0_darwin_arm64.tar.gz
$ tar -xzf terraform-provider-msl_0.1.0_darwin_arm64.tar.gz
Archive:  terraform-provider-msl_0.1.0_darwin_arm64.tar.gz
  inflating: CHANGELOG.md
  inflating: LICENSE
  inflating: README.md
  inflating: terraform-provider-msl_v0.1.0
```

### 2. Configure the dev override

In `examples/dev.terraformrc`, adjust the path to point to directory containing the provider binary (e.g. `/tmp/terraform-provider-msl`).

```hcl
provider_installation {
  dev_overrides {
    "harmonicinc-video/msl" = "/tmp/terraform-provider-msl"
  }
  direct {}
}
```

### 3. Configure the provider and resources

Create a Terraform configuration (main.tf) that references the provider.

```hcl
terraform {
  required_providers {
    msl = {
      source  = "harmonicinc-video/msl"
      version = "~> 0.1"
    }
  }
}

provider "msl" {
  endpoint  = var.api_endpoint   # e.g. "https://api.msl.example.com"
  api_token = var.api_token      # sensitive — use tfvars or env
}
```

See [`examples/quick-start/main.tf`](examples/quick-start/main.tf) for a complete working example.

### 4. Manage an Origin

Defines your desired MSL5 resources. This will create an Origin with the specified properties during `terraform apply`. Adjust the properties as needed.

```hcl
resource "msl_origin" "terraformtest" {
  host_name       = "uswestterraformtest"
  ingest_location = "US_SEA"
  contract_id     = "0"
  cptag           = "0"

  group_id = "dummy"
}

output "origin_id" { value = msl_origin.terraformtest.id }
```

### 5. Run the demo

Under Repository Root:

```bash
$ cd examples/quick-start
$ cp terraform.tfvars.example terraform.tfvars
# edit terraform.tfvars with real values
# -- OR -- set variables via TF_VAR_ environment variables:
# export TF_VAR_api_token="<your-api-token>"
# Note: terraform.tfvars takes precedence over TF_VAR_ env vars. Remove a
# variable from terraform.tfvars for the corresponding env var to take effect.

$ TF_CLI_CONFIG_FILE=../dev.terraformrc terraform refresh

$ TF_CLI_CONFIG_FILE=../dev.terraformrc terraform plan -out=tfplan
# review the plan output to verify the expected changes

$ TF_CLI_CONFIG_FILE=../dev.terraformrc terraform apply tfplan
# review the apply output to verify the expected changes, then confirm

$ TF_CLI_CONFIG_FILE=../dev.terraformrc terraform show
# view current state and computed attributes

$ TF_CLI_CONFIG_FILE=../dev.terraformrc terraform destroy
# review the destroy output to verify the expected changes, then confirm
```

## Building the Provider

If you want to build the provider from source, clone the repository and run:

```bash
git clone https://github.com/harmonicinc-video/terraform-provider-msl.git
cd terraform-provider-msl
make build
make install   # copies binary to ~/.terraform.d/plugins/
```

### Development

Available Make Targets:

```bash
make build    # compile
make install  # install to local plugin dir
make test     # run unit tests
make fmt      # format code + HCL
make lint     # go vet
```

## Debugging

To enable verbose HTTP logging from the provider, set `debug_logging = true` in the provider block:

```hcl
provider "msl" {
  endpoint      = var.api_endpoint
  api_token     = var.api_token
  debug_logging = true
}
```

Then run Terraform with `TF_LOG=DEBUG` to surface the `[DEBUG] msl:` log lines:

```bash
TF_LOG=DEBUG TF_CLI_CONFIG_FILE=../dev.terraformrc terraform plan  -out=tfplan
```

> **Note:** `debug_logging` must be `true` in the provider config — `TF_LOG=DEBUG` alone is not enough to emit MSL HTTP logs.

## Provider Configuration Reference

See [docs/index.md](docs/index.md) for the full provider schema.
