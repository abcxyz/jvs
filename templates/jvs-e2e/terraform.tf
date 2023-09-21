locals {
  project_id = "REPLACE_PROJECT_ID"
}

module "jvs_e2e" {
  source = "git::https://github.com/abcxyz/jvs.git//terraform/e2e?ref=REPLACE_JVS_SHA"

  project_id = local.project_id

  jvs_container_image = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/jvsctl:REPLACE_JVS_CONTAINER_IMAGE_TAG"
  jvs_api_domain      = "REPLACE_JVS_API_DOMAIN"
  jvs_ui_domain       = "REPLACE_JVS_UI_DOMAIN"
  iap_support_email   = "REPLACE_IAP_SUPPORT_EMAIL"

  // Alerting is disabled but we still need an email.
  notification_channel_email = "REPLACE_NOTIFICATION_CHANNEL_EMAIL"

  // Use gcloud app id because Cloud Run accepts it.
  prober_audience  = "REPLACE_PROBER_AUDIENCE"
  jvs_prober_image = "us-docker.pkg.dev/abcxyz-artifacts/docker-images/jvs-prober:REPLACE_JVS_PROBER_IMAGE_TAG"
}