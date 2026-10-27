# Carbyne Stack - Intel SGX TEE Support for Offline Phase

This document provides step-by-step instructions for setting up Carbyne Stack
with TEE (Trusted Execution Environment) for offline phase.

> **Note**: This implementation has only been tested on a Azure Kubernetes
> Service (AKS). The instructions may contain commands specific to Azure CLI,
> but it can be executed on other CSPs provided they are Kubernetes clusters
> and the respective SGX plugins are enabled.

## Overview

The Intel SGX TEE integration with Carbyne Stack enables secure, hardware-backed
execution of the Correlated Randomness Generation (CRG) process within Intel SGX
enclaves. The implementation uses Gramine (a library OS for SGX) to run the
MP-SPDZ Fake-Offline application inside protected enclaves, providing
confidentiality and integrity guarantees for the offline phase of Secure
Multiparty Computation.

The CRG application implements the Klyshko Integration Interface (KII) and
performs local attestation between enclaves on the same node, followed by
remote attestation across VCPs using RA-TLS (Remote Attestation TLS) with DCAP.
MAC key shares are securely exchanged through protobuf messages over attested
TLS connections, ensuring that tuple generation occurs only after successful
attestation of all participating enclaves.

## Prerequisites

- Carbyne Stack v0.8.0
  ([CS Prerequisites](https://carbynestack.io/documentation/getting-started/prerequisites/#prerequisites))
- Two VCPs setup as per Carbyne Stack Platform Setup
- Ensure that the node has SGX hardware and AESMD service daemon is present.
  For example, below command shows how to create a node on Azure with SGX

```bash
az aks nodepool add \
  --resource-group "${azResourceGroup:?}" \
  --name nodepool2 \
  --cluster-name "${azClusterName:?}" \
  --node-count 1 \
  --os-sku Ubuntu \
  --node-vm-size Standard_Dc2s_v3 \
  --enable-node-public-ip
```

## Deployment Steps

## Step 1: Creating AKS Cluster with SGX Support

Azure-specific SGX device plugin is required, which is only available in AKS
cluster as a confcom AKS addon.

**Important**: Before SGX plugin installation, ensure the "confcom" plugin is
enabled in the resource group.

The `sgx-quotehelper` plugin must be installed in both VCP 1 and VCP 2. The
commands listed below are for VCP1. These commands are to be executed on the
Azure Admin Console.

### VCP-1 Setup

```bash
export azResourceGroup=YOUR_RESOURCE_GROUP
export azClusterName=YOUR_CLUSTER_NAME

# Create AKS cluster with confcom addon
az aks create \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}" \
  --kubernetes-version 1.28 \
  --os-sku Ubuntu \
  --node-vm-size Standard_D2as_v4 \
  --node-count 1 \
  --generate-ssh-keys \
  --enable-addons confcom

# Get cluster credentials
az aks get-credentials \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}"

# Add SGX-enabled nodepool
az aks nodepool add \
  --resource-group "${azResourceGroup:?}" \
  --name nodepool2 \
  --cluster-name "${azClusterName:?}" \
  --node-count 1 \
  --os-sku Ubuntu \
  --node-vm-size Standard_Dc2s_v3 \
  --enable-node-public-ip

# Enable sgx-quotehelper
az aks addon update \
  --addon confcom \
  --resource-group "${azResourceGroup:?}" \
  --name "${azClusterName:?}" \
  --enable-sgxquotehelper
```

**Reference**: [Azure Confidential Computing - Confidential Enclave Nodes AKS
Get Started](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-enclave-nodes-aks-get-started)

### Validate SGX Plugin Installation

Validate that the SGX plugin and SGX quote helper are running. If the below
mentioned pods are running, then the plugins are enabled successfully.

```bash
kubectl get pods -n kube-system | grep sgx
```

Expected output of above command:

```bash
sgx-plugin-64nnt            1/1   Running   0   42m
sgx-quote-helper-9gnrf      1/1   Running   0   17m
sgx-webhook-6cdb9994c5-6jhw4 1/1   Running   0   42m
```

## Step 2: SGX Security Configurations

Gramine signs the SGX enclave with an RSA private key. The hash of the
corresponding public key is reported by the SGX hardware as `MR_SIGNER` during
attestation and identifies who built the enclave.

### Official images

Official images are signed with the Carbyne Stack author key. This key is
never part of the repository or the image; it is only available to the release
workflow. Its public key is published in [author-key.pub.pem](author-key.pub.pem)
and the resulting `MR_SIGNER` value in [MR_SIGNER](MR_SIGNER). If you use the
official image, no key is required on your side. See
[Verifying official images](#verifying-official-images) for how to check an
image before deploying it.

### Building your own image (development)

For local development, generate your own throwaway signing key. SGX requires a
3072-bit RSA key with public exponent 3 (`-3`):

```bash
openssl genrsa -3 -out enclave-key.pem 3072
```

The key is passed to the build as a BuildKit secret, so it is used for signing
but never stored in an image layer:

```bash
docker build --secret id=enclave_key,src=enclave-key.pem \
  -f Dockerfile.tee-fake-offline -t klyshko-mp-spdz-tee:local-dev .
```

> **Important:**\
> Never commit `enclave-key.pem` or copy it into the build context of an image
> you publish. Images built with your own key have a different `MR_SIGNER` and
> do not attest against official images.

For more information refer the
[Gramine Docs](https://gramine.readthedocs.io/en/stable/python/writing-sgx-sign-plugins.html)

### RA-TLS and KII environment variables

- **RA-TLS**: `RA_TLS_ALLOW_DEBUG_ENCLAVE_INSECURE` (default `0`),
  `RA_TLS_ALLOW_OUTDATED_TCB_INSECURE` (default `0`),
  `RA_TLS_ALLOW_HW_CONFIG_NEEDED` (default `1`),
  `RA_TLS_ALLOW_SW_HARDENING_NEEDED` (default `1`). Only change the first two
  for debug or relaxed TCB policies (insecure).
- **MRSIGNER**: `KII_MRSIGNER` defaults to signer value from sigstruct. export
  KII_MRSIGNER=0 to disable MRSIGNER verification.

## Verifying official images

Official images are built, pushed and signed by the
`mp-spdz-tee.publish.yaml` workflow of this repository using
[Sigstore Cosign](https://docs.sigstore.dev/) keyless signing. Each image
carries a signed attestation that binds the image digest to its enclave
measurements (`MR_ENCLAVE`, `MR_SIGNER`).

```bash
IMG=ghcr.io/carbynestack/klyshko-mp-spdz-tee@sha256:<digest>
ID='^https://github\.com/carbynestack/klyshko/\.github/workflows/mp-spdz-tee\.publish\.yaml@refs/tags/mp-spdz-tee-v[0-9]+\.[0-9]+\.[0-9]+$'
ISSUER=https://token.actions.githubusercontent.com

# 1. Image was built by the official release workflow
cosign verify "$IMG" \
  --certificate-identity-regexp "$ID" \
  --certificate-oidc-issuer "$ISSUER"

# 2. Signed enclave measurements of this image
cosign verify-attestation "$IMG" \
  --type https://carbynestack.io/attestations/gramine-sgx-measurements/v1 \
  --certificate-identity-regexp "$ID" \
  --certificate-oidc-issuer "$ISSUER" \
  | jq -r .payload | base64 -d | jq .predicate
```

Check that:

- `mr_signer` equals the value in [MR_SIGNER](MR_SIGNER),
- `debug_enclave` is `false`, and
- `mr_enclave` matches what the running enclaves report (printed in the
  `player_*.log` files of the generator pods).

> **Tip**: If you only know the version tag, `cosign verify` also accepts
> `ghcr.io/carbynestack/klyshko-mp-spdz-tee:<version>`. It verifies the digest
> the tag currently points to and prints it; use that digest for the
> attestation check.

## Step 3: Bringing Up Carbyne Stack

### Platform Setup

[Carbyne Stack Platform Setup](https://github.com/carbynestack/carbynestack.github.io/blob/20abf2f7a45c4840a5bde5bb6e0c93516732fc24/docs/documentation/getting-started/deployment/manual/platform-setup.md)

> **Note**: If you are using a managed K8 cluster like AKS, you can skip
> installation of `kind`. If the managed cluster also has its own load
> balancer, `MetalLB` installation can be skipped.

### Stack Setup

Install the Stack by following the steps in
[Carbyne Stack Setup](https://carbynestack.io/documentation/getting-started/deployment/manual/stack/).
Follow all steps till Step 4: Configure the Correlated Randomness Generator
(CRG) used by Klyshko. Instead of using the default insecure CRG, we will
configure carbyne stack to run TEE-Accelerated Tuple Generation.

#### Using the official image (recommended)

Point the generator to the official TEE image. Replace `<version>` with the
release you verified in
[Verifying official images](#verifying-official-images):

```bash
export KLYSHKO_GENERATOR_IMAGE_REPOSITORY="carbynestack/klyshko-mp-spdz-tee"
export KLYSHKO_GENERATOR_IMAGE_TAG="<version>"
```

The official image is published on `ghcr.io`, which is the default registry for
all Klyshko images, so no further registry configuration is needed.

#### Using a custom image

If you built your own image (see
[Building your own image](#building-your-own-image-development)) and pushed it
to a different registry, set:

```bash
export KLYSHKO_IMAGE_REGISTRY="docker.io"
export KLYSHKO_GENERATOR_IMAGE_REPOSITORY="LINK_TO_DOCKER_REPO"
export KLYSHKO_GENERATOR_IMAGE_TAG="YOUR_TAG"
```

**Provisioner Image Registry Fix**: `KLYSHKO_IMAGE_REGISTRY` is also used for
the operator and provisioner images. To keep pulling those from the default
registry while using a custom generator registry, edit the klyshko helmfile in
the `carbynestack/deployments` folder:

```bash
vi helmfile.d/0400.klyshko.yaml
```

Change this line under `release -> values -> provisioner -> image` and
`release -> values -> controller -> image`:

```yaml
registry: {{ env "KLYSHKO_IMAGE_REGISTRY" | default "ghcr.io" }}
```

To:

```yaml
registry: {{ env "KLYSHKO_PROVISIONER_IMAGE_REGISTRY" | default "ghcr.io" }}
# in provisioner

registry: {{ env "KLYSHKO_OPERATOR_IMAGE_REGISTRY" | default "ghcr.io" }}
# in controller 
```

These will pull required docker images for TEE acceleration and use default
registries for klyshko operator and provisioners.

## Step 4: Enabling SGX in Klyshko Config

After the pods have been deployed in the two clusters, use --sgx-enabled flag.
This will enable the SGX mode in the klyshko operator.

```bash
kubectl patch deployment klyshko-controller-manager -n default --type='json' \
  -p='[{"op":"add","path":"/spec/template/spec/containers/1/args/-",'\
'"value":"--sgx-enabled"}]'
```

## Troubleshooting

### SGX Pods Not Running

If SGX plugin pods are not running, check:

1. Node selector labels match the actual node instance type (Refer below for
   fix)
2. Custom daemonsets are deployed correctly
3. Node has SGX support enabled

### Node Selector Label Mismatch Fix

In new clusters with addons enabled, the `sgx-plugin` and `sgx-quote-helper`
pods might not run due to a mismatch in node-selector labels
`node.kubernetes.io/instance-type` between the machine and the daemonset
addon.

**Fix Steps**:

1. Get manifests for sgx-quote-helper and sgx-plugin:

   ```bash
   kubectl get daemonsets.apps -n kube-system sgx-quote-helper -o yaml \
     > sgx-quote-helper-custom.yaml
   kubectl get daemonsets.apps -n kube-system sgx-plugin -o yaml \
     > sgx-plugin-custom.yaml
   ```

2. Edit each manifest:
   - Change `node.kubernetes.io/instance-type` from `Standard_DC_*` to
     `Standard_Dc_*`
   - Remove `last-applied-configuration` annotation
   - Rename daemonset app labels to `sgx-plugin-custom` and
     `sgx-quote-helper-custom` (this prevents Azure from reverting changes)

3. Deploy the custom yamls:

   ```bash
   kubectl apply -f sgx-quote-helper-custom.yaml
   kubectl apply -f sgx-plugin-custom.yaml
   ```

Repeat the same steps for VCP-2 with appropriate resource group and cluster
name.

### etcd Connection Issues

If etcd instances cannot connect:

1. Verify `KLYSHKO_ETCD_ENDPOINT` is set correctly with IP and port
2. Check firewall rules allow communication between clusters
3. Ensure both clusters can reach each other's etcd endpoints

### Image Pull Errors

If images fail to pull:

1. Verify image registry credentials are configured
2. If you use a custom generator registry, check that the provisioner registry
   fix is applied
3. Ensure all required environment variables are exported

### Signature Verification Fails

If `cosign verify` or `cosign verify-attestation` fails:

1. Make sure you are verifying an image from
   `ghcr.io/carbynestack/klyshko-mp-spdz-tee`, not a custom build
2. Check that you copied the `ID` and `ISSUER` values exactly as shown above
3. Do not deploy the image until verification succeeds
