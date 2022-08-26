# JVS Setup

**JVS is not an official Google product.**

## Prerequisites
you must have:
* An existing cloud org
* A billing account you can use in the cloud org
* A project you can use in the cloud org

1. Install [gcloud](https://cloud.google.com/sdk/docs/install)
2. make sure you are logged in with gcloud.
   ```shell
   gcloud auth login --update-adc
   ```
3. Install [grpcurl](https://github.com/fullstorydev/grpcurl)

## Set Up

1. Change directory to where terraform code lives
   ```shell
   cd terraform
   ```
2. Copy an existing environment (e.g. dev)
   ```shell
   cp -r dev my-env && cd my-env && rm .terraform.lock.hcl
   ```
3. When you create a new configuration
   or check out an existing configuration from version control,
   you need to initialize the directory with:
   ```shell
   terraform init
   ```
4. Time to apply
   ```shell
   terraform apply
   ```
   If you get a message like `The GCP project to host the justification verification service`,
   please enter the GCP project where you want the JVS system gets installed.
5. Wait until it’s done then you have a test environment up;
   there will be a few outputs which you need to remember for later use.
   You will see output similar to follows
```shell
Outputs:

cert_rotator_server_url = "https://cert-rotator-e2e-xxxxx-uc.a.run.app"
jvs_server_url = "https://jvs-e2e-xxxx-uc.a.run.app"
public_key_server_url = "https://pubkey-e2e-xxxx-uc.a.run.app"