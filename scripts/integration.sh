#!/bin/bash
# Copyright 2023 The Authors (see AUTHORS file)
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eEuo pipefail

SERVICES_TF_MODULE_DIR="./terraform/modules/jvs-services"
SERVICES_VAR_FILE="/tmp/jvs_ci.tfvars"

# If BUILD_COMMON is true, build common cloud infra including cloud run service
# accounts and KMS keyring.
if $BUILD_COMMON
then
  COMMON_TF_MODULE_DIR="./terraform/modules/common"
  COMMON_VAR_FILE="/tmp/jvs_common.tfvars"
  COMMON_TF_MODULE_DIR="./terraform/modules/common"
  COMMON_VAR_FILE="/tmp/jvs_common.tfvars"

  touch /tmp/jvs_common.tfvars

  clean_up_common() {
    terraform -chdir=$COMMON_TF_MODULE_DIR destroy -auto-approve -var-file=$COMMON_VAR_FILE
    rm $COMMON_VAR_FILE
  }

  trap clean_up_common EXIT

  echo project_id=\"$PROJECT_ID\" >> $COMMON_VAR_FILE;

  terraform -chdir=$COMMON_TF_MODULE_DIR init
  terraform -chdir=$COMMON_TF_MODULE_DIR apply -auto-approve -var-file=$COMMON_VAR_FILE

  API_SA=$(terraform -chdir=${COMMON_TF_MODULE_DIR} output -raw jvs_api_service_account_email)
  UI_SA=$(terraform -chdir=${COMMON_TF_MODULE_DIR} output -raw jvs_ui_service_account_email)
  CERT_ROTATOR_SA=$(terraform -chdir=${COMMON_TF_MODULE_DIR} output -raw jvs_cert_rotator_service_account_email)
  PUBLIC_KEY_SA=$(terraform -chdir=${COMMON_TF_MODULE_DIR} output -raw jvs_public_key_service_account_email)
  KMS_KEYRING_ID=$(terraform -chdir=${COMMON_TF_MODULE_DIR} output -raw kms_keyring_id)
fi

touch /tmp/jvs_ci.tfvars

clean_up_services() {
  terraform -chdir=$SERVICES_TF_MODULE_DIR destroy -auto-approve -var-file=$SERVICES_VAR_FILE
  rm $SERVICES_VAR_FILE
}

trap clean_up_services EXIT

echo project_id=\"$PROJECT_ID\" >> $SERVICES_VAR_FILE;
echo jvs_api_service_account=\"$API_SA\" >> $SERVICES_VAR_FILE;
echo jvs_ui_service_account=\"$UI_SA\" >> $SERVICES_VAR_FILE;
echo jvs_cert_rotator_service_account=\"$CERT_ROTATOR_SA\" >> $SERVICES_VAR_FILE;
echo jvs_public_key_service_account=\"$PUBLIC_KEY_SA\" >> $SERVICES_VAR_FILE;
echo jvs_api_service_image=\"${REGISTRY}/jvs-justification:${DOCKER_TAG}\" >> $SERVICES_VAR_FILE;
echo jvs_ui_service_image=\"${REGISTRY}/jvs-ui:${DOCKER_TAG}\" >> $SERVICES_VAR_FILE;
echo jvs_cert_rotator_service_image=\"${REGISTRY}/jvs-cert-rotation:${DOCKER_TAG}\" >> $SERVICES_VAR_FILE;
echo jvs_public_key_service_image=\"${REGISTRY}/jvs-public-key:${DOCKER_TAG}\" >> $SERVICES_VAR_FILE;
echo kms_keyring_id=\"$KMS_KEYRING_ID\" >> $SERVICES_VAR_FILE;
echo kms_key_name=\"jvs-key-$RANDOM\" >> $SERVICES_VAR_FILE;
# Skip cloud scheduler creation.
echo kms_key_rotation_minutes=0 >> $SERVICES_VAR_FILE;
echo public_key_invokers=[] >> $SERVICES_VAR_FILE;

terraform -chdir=$SERVICES_TF_MODULE_DIR init
terraform -chdir=$SERVICES_TF_MODULE_DIR apply -auto-approve -var-file=$SERVICES_VAR_FILE

# TODO(#158): add real service integration test.