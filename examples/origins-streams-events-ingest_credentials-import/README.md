This example demonstrates how to import existing origins, streams, events, and ingest credentials into Terraform state using the MSL Terraform provider.

Define each resource in `terraform.tfvars` (using `terraform.tfvars.example` as a template), then run the matching `terraform import` commands to bring existing resources under Terraform management. Resources must be imported in dependency order: origins first, then streams, then events and ingest credentials.

## 1. Prepare your tfvars

Copy `terraform.tfvars.example` to `terraform.tfvars` and fill in the actual values that match the existing resources in the API.

## 2. Import origins

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform import \
  'msl_origin.managed["uswestterraformtest"]' \
  "<origin-id>"
```

## 3. Import streams

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform import \
  'msl_stream.managed["uswestterraformtest_hls"]' \
  "<stream-id>"
```

## 4. Import events

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform import \
  'msl_event.managed["uswestterraformtest_event"]' \
  "<stream-id>/<event-name>"
```

## 5. Import ingest credentials

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform import \
  'msl_ingest_credential.managed["encoder_live"]' \
  "<stream-id>/<credential-id>"
```

## 6. Verify and reconcile

After all imports, inspect the state and reconcile any drift:

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform state show 'msl_origin.managed["uswestterraformtest"]'
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform state show 'msl_stream.managed["uswestterraformtest_hls"]'
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform state show 'msl_event.managed["uswestterraformtest_event"]'
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform state show 'msl_ingest_credential.managed["encoder_live"]'
```

Run `terraform plan` to confirm there are no unintended changes before applying:

```bash
TF_CLI_CONFIG_FILE=../dev.terraformrc terraform plan -out=tfplan
```
