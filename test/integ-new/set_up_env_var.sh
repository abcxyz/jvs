#!/usr/bin/env bash
export TEST_INTEGRATION=true
export API_SERVICE_NAME=jvs-api-4527
export PUBLIC_KEY_SERVICE_NAME=jvs-public-key-f22e
export SERVICES_URL_POSTFIX=2nhpyabgtq-uc.a.run.app
export TAG_ID=ci-6177333825-7
export INTEG_TEST_ID_TOKEN=$(gcloud auth print-identity-token)
export INTEG_TEST_WIF_SERVICE_ACCOUNT=qinhang@google.com
export INTEG_TEST_JWKS_ENDPOINT=https://ci-6177333825-7---jvs-public-key-f22e-2nhpyabgtq-uc.a.run.app/.well-known/jwks
export INTEG_TEST_API_SERVER=ci-6177333825-7---jvs-api-4527-2nhpyabgtq-uc.a.run.app:443
