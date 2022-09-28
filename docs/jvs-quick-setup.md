# JVS Setup

**JVS is not an official Google product.**

## Prerequisites

you must have:

*   An existing cloud org
*   A billing account you can use in the cloud org
*   A project you can use in the cloud org

1.  Install [gcloud](https://cloud.google.com/sdk/docs/install)
2.  make sure you are logged in with gcloud.

    ```shell
    gcloud auth login --update-adc
    ```

3.  Install [jvsctl](cli-tool.md/#install)

## Set Up

1.  Change directory to where terraform code lives

    ```shell
    cd terraform
    ```

2.  Copy an existing environment (e.g. quick-setup)

    ```shell
    cp -r quick-setup my-env && cd my-env
    ```

3.  When you create a new configuration or check out an existing configuration
    from version control, you need to initialize the directory with:

    ```shell
    terraform init
    ```

4.  Time to apply

    ```shell
    terraform apply
    ```

    If you get a message like `The GCP project to host the justification
    verification service`, please enter the GCP project where you want the JVS
    system gets installed.

5.  Wait until it’s done then you have a test environment up; there will be a
    few outputs which you need to remember for later use. You will see output
    similar to follows

    ```shell
    Outputs:
    cert_rotator_server_url ="https://cert-rotator-e2e-xxxxx-uc.a.run.app"
    jvs_server_url ="https://jvs-e2e-xxxx-uc.a.run.app"
    public_key_server_url ="https://pubkey-e2e-xxxx-uc.a.run.app"
    ```

Besides above servers, [KMS](https://cloud.google.com/security-key-management)
resources e.g.
[keyRing](https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings)
and
[cryptoKey](https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys)
will also get created.

## Try JVS APIs

### Justification API

1.  Export the jvs server with the domain part of the `jvs_server_url` from
    Terraform outputs like `jvs-e2e-xxxx-uc.a.run.app`, and export the JWKS
    endpoint with `public_key_server_url` from Terraform outputs if you want
    to validate token via CLI.

    ```shell
    export SERVER=<jvs_server_domain>:443
    export JWKS_ENDPOINT=public_key_server_url
    ```

2.  Create Justification Token via [jvsctl](cli-tool.md):

    ```shell
    jvsctl token --explanation "issues/12345" --ttl 30m --server ${SERVER}
    ```

3.  Validate Justification [jvsctl](cli-tool.md):

    ```shell
    jvsctl validate --token "example token" --jwks_endpoint ${JWKS_ENDPOINT}

    # or pass token via pipe
    echo "${JVS_TOKEN}" | jvsctl validate --token -
    cat /tmp/jvs_token | jvsctl validate --token -
    ```

### Public Key API

1.  Export the `public_key_server_url` from Terraform outputs

    ```shell
    export PUBLIC_KEY_SERVER_URL=<public_key_server_url>
    ```

2.  Fetch public keys via command:

    ```shell
    curl -H "Authorization: Bearer $(gcloud auth print-identity-token )" \
    "${PUBLIC_KEY_SERVER_URL}/.well-known/jwks"
    ```

    You should see output similar to follows

    ```shell
    {"keys":[{"crv":"P-256","kid":"projects/test-project/locations/global/keyRings/ci-keyring/cryptoKeys/jvs-key/cryptoKeyVersions/1",
    "kty":"EC","x":"u4SVWCYAZtD8J9r4bc150doTctTviIltS215qKkw8bF","y":"E3zbf_rvi7jTQykxcyUZqerXo_ssS6auvwR6mLchLll"},
    {"crv":"P-256","kid":"projects/test-project/locations/global/keyRings/ci-keyring/cryptoKeys/jvs-key/cryptoKeyVersions/2",
    "kty":"EC","x":"L4tcY2n2qKngEsLzatLXE_iTK39hUg18bE27H-r_p_M","y":"S0TrLBOPhyw7guoEIR2LSU6tLhelHLE3pZ4XaEJnzLN"}]}
    ```

### Cert Rotation API

1.  Export the `cert_rotator_server_url` from Terraform outputs

    ```shell
    export CERT_ROTATOR_SERVER_URL=<cert_rotator_server_url>
    ```

2.  Rotate keys via command:

    ```shell
    curl -H "Authorization: Bearer $(gcloud auth print-identity-token )" \
    "${CERT_ROTATOR_SERVER_URL}"
    ```

    You should see output similar to follows

    ```shell
    finished with all keys successfully.
    ```

## Clean up

Run this command to tear down resources:

```shell
terraform destroy
```
