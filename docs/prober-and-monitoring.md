# Prober and Alerting

**JVS is not an official Google product.**

## Prober

[Prober](../prober/) is a bash script that uses [jvsctl](../cmd/jvsctl/) to request and validate tokens. 

In each prober job, the prober will request and validate a token with jvs service. The job will be treated as a success only if the following requirement is met.
  1. Request a token successfully.
  2. Validate the token successfully.

The prober will be deployed using [GCP's cloud run job](https://cloud.google.com/run/docs/create-jobs), and use [cloud scheduler](https://cloud.google.com/scheduler/docs/overview) to trigger the cloud run job on a user defined frequency.

## Monitoring and Alert Policy

We monitor all JVS services with [native cloud run monitoring metrics](https://cloud.google.com/monitoring/api/metrics_gcp#gcp-run) and alert based on the following metrics:

-  Request Count 
-  Request Latency

And we also create alert policy for each JVS service on the above two metrics.

Similarly, we alert base on cloud run job execution metrics for prober service.
See alert policies in [prober.tf](../terraform/modules/monitoring/prober.tf).

## Installation

You can use the provided Terraform module to setup Prober, or you can refer to the provided module to build it from scratch.

```
module "jvs_monitoring" {
  source = "git::https://github.com/abcxyz/jvs.git//terraform/modules/monitoring?ref=main"

  project_id = "YOUR PROJECT ID"

  jvs_service_name               = "Name of your jvs-api service"
  cert_rotator_service_name      = "Name of your cert rotator service"
  public_key_service_name        = "Name of your public key service"
  jvs_ui_service_name            = "Name of your ui-service service"
  notification_channel_email     = "Email to which alert will be sent to"
  prober_jvs_api_address         = "https://jvs.corp.internal:8080"
  prober_jvs_public_key_endpoint = "https://keys.corp.internal:8080/.well-known/jwks"
  jvs_prober_image               = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/jvs-prober:0.0.5-amd64"
  prober_audience                = "This would be either your gcloud app id, or jvs-api service's url"
}
```

By default, alering is disabled, you can enable it by setting the following variables:
```
alert_enabled        = true

```

You can also change threshold to your desired value. And example would be:
```
cert_rotator_5xx_response_threshold = 10
cert_rotator_latency_threshold_ms   = 10000
```

To add more alerting policies for JVS services, you can do so by adding terraform code to [alert.tf](../terraform/modules/monitoring/alert.tf)

