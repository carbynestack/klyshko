# Carbyne Stack - Intel SGX TEE Support for Offline Phase

This document provides step-by-step instructions for setting up Carbyne Stack with TEE (Trusted Execution Environment) for offline phase.

> **Note**: This implementation has only been tested on a Azure Kubernetes Service (AKS). The instructions may contain commands specific to Azure CLI, but it can be executed on other CSPs provided they are Kubernetes clusters and the respective SGX plugins are enabled.

## Overview

The Intel SGX TEE integration with Carbyne Stack enables secure, hardware-backed execution of the Correlated Randomness Generation (CRG) process within Intel SGX enclaves. The implementation uses Gramine (a library OS for SGX) to run the MP-SPDZ Fake-Offline application inside protected enclaves, providing confidentiality and integrity guarantees for the offline phase of Secure Multiparty Computation. 

The CRG application implements the Klyshko Integration Interface (KII) and performs local attestation between enclaves on the same node, followed by remote attestation across VCPs using RA-TLS (Remote Attestation TLS) with DCAP. MAC key shares are securely exchanged through protobuf messages over attested TLS connections, ensuring that tuple generation occurs only after successful attestation of all participating enclaves.

## Prerequisites

- Carbyne Stack v0.8.0 ([CS Prerequisites](https://carbynestack.io/documentation/getting-started/prerequisites/#prerequisites))
- Two VCPs setup as per Carbyne Stack Platform Setup
- Ensure that the node has SGX hardware and AESMD service daemon is present. For example, below command shows how to create a node on Azure with SGX

```az aks nodepool add   --resource-group "${azResourceGroup:?}"   --name nodepool2   --cluster-name "${azClusterName:?}"   --node-count 1   --os-sku Ubuntu   --node-vm-size Standard_Dc2s_v3	--enable-node-public-ip```

## Deployment Steps

## Step 1: Creating AKS Cluster with SGX Support

Azure-specific SGX device plugin is required, which is only available in AKS cluster as a confcom AKS addon.

**Important**: Before SGX plugin installation, ensure the "confcom" plugin is enabled in the resource group.

The `sgx-quotehelper` plugin must be installed in both VCP 1 and VCP 2. The commands listed below are for VCP1. These commands are to be executed on the Azure Admin Console.

#### VCP-1 Setup
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

**Reference**: [Azure Confidential Computing - Confidential Enclave Nodes AKS Get Started](https://learn.microsoft.com/en-us/azure/confidential-computing/confidential-enclave-nodes-aks-get-started)

#### Validate SGX Plugin Installation

Validate that the SGX plugin and SGX quote helper are running. If the below mentioned pods are running, then the plugins are enabled succesfully. 

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

To enable attestation of Intel SGX TEE, the private key (enclave-key.pem) must be set, which will be used by Gramine to sign the hardware.

To generate a new key, use open ssl.
```
openssl genrsa -out enclave-key.pem 3072
```

The same enclave-key.pem must be present in both the VCPs, else the remote attestation will fail.

For more information refer the [Gramine Docs](https://gramine.readthedocs.io/en/stable/python/writing-sgx-sign-plugins.html)


> **Note**: In kii-run.sh, the flags RA_TLS_ALLOW_DEBUG_ENCLAVE_INSECURE and RA_TLS_ALLOW_OUTDATED_TCB_INSECURE have been enabled for development and testing purposes.

For production deployments, set these flags to 0 before building the image.

```
export RA_TLS_ALLOW_DEBUG_ENCLAVE_INSECURE=0
export RA_TLS_ALLOW_OUTDATED_TCB_INSECURE=0
```


## Step 3: Bringing Up Carbyne Stack

### Platform Setup

[Carbyne Stack Platform Setup](https://github.com/carbynestack/carbynestack.github.io/blob/20abf2f7a45c4840a5bde5bb6e0c93516732fc24/docs/documentation/getting-started/deployment/manual/platform-setup.md)

> **Note**: If you are using a managed K8 cluster like AKS, you can skip installation of `kind`. If the managed cluster also has it's own load balancer, `MetalLB` installation can be skipped.


### Stack Setup

Install the Stack by following the steps in [Carbyne Stack Setup](https://carbynestack.io/documentation/getting-started/deployment/manual/stack/). Follow all steps till Step 4: Configure the Correlated Randomness Generator (CRG) used by Klyshko. Instead of using the default insecure CRG, we will configure carbyne stack to run TEE-Accelerated Tuple Generation.

Build the image of the [TEE Generator](https://github.com/datakaveri/klyshko/blob/merged-cs-tee/klyshko-mp-spdz-tee/Dockerfile.tee-fake-offline)

```bash
export KLYSHKO_IMAGE_REGISTRY="docker.io"
export KLYSHKO_GENERATOR_IMAGE_REPOSITORY="LINK_TO_DOCKER_REPO"
```

**Provisioner Image Registry Fix**: We only intend to use a custom generator from docker.io, however the operator and provisioner images can be default. To ensure this, we must make a change in the klyshko helmfile. In the `carbynestack/deployments` folder, edit this file:

```bash
vi helmfile.d/0400.klyshko.yaml
```

Change this line under `release -> values -> provisioner -> image` and `release -> values -> controller -> image`:

```yaml
registry: {{ env "KLYSHKO_IMAGE_REGISTRY" | default "ghcr.io" }}
```

To:

```yaml
registry: {{ env "KLYSHKO_PROVISIONER_IMAGE_REGISTRY" | default "ghcr.io" }} # in provisioner

registry: {{ env "KLYSHKO_OPERATOR_IMAGE_REGISTRY" | default "ghcr.io" }} # in controller 
```

These will pull required docker images for TEE acceleration and use default registries for klyshko operator and provisioners.

## Step 3: Enabling SGX in Klyshko Config

After the pods have been deployed in the two clusters, use --sgx-enabled flag. This will enable the SGX mode in the klyshko operator.

```
kubectl patch deployment klyshko-controller-manager -n default --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/1/args/-","value":"--sgx-enabled"}]'
```

## Troubleshooting

### SGX Pods Not Running

If SGX plugin pods are not running, check:
1. Node selector labels match the actual node instance type (Refer below for fix)
2. Custom daemonsets are deployed correctly
3. Node has SGX support enabled

### Node Selector Label Mismatch Fix

In new clusters with addons enabled, the `sgx-plugin` and `sgx-quote-helper` pods might not run due to a mismatch in node-selector labels `node.kubernetes.io/instance-type` between the machine and the daemonset addon.

**Fix Steps**:

1. Get manifests for sgx-quote-helper and sgx-plugin:

```bash
kubectl get daemonsets.apps -n kube-system sgx-quote-helper -o yaml > sgx-quote-helper-custom.yaml
kubectl get daemonsets.apps -n kube-system sgx-plugin -o yaml > sgx-plugin-custom.yaml
```

2. Edit each manifest:
   - Change `node.kubernetes.io/instance-type` from `Standard_DC_*` to `Standard_Dc_*`
   - Remove `last-applied-configuration` annotation
   - Rename daemonset app labels to `sgx-plugin-custom` and `sgx-quote-helper-custom` (this prevents Azure from reverting changes)

3. Deploy the custom yamls:

```bash
kubectl apply -f sgx-quote-helper-custom.yaml
kubectl apply -f sgx-plugin-custom.yaml
```

Repeat the same steps for VCP-2 with appropriate resource group and cluster name.

### etcd Connection Issues

If etcd instances cannot connect:
1. Verify `KLYSHKO_ETCD_ENDPOINT` is set correctly with IP and port
2. Check firewall rules allow communication between clusters
3. Ensure both clusters can reach each other's etcd endpoints

### Image Pull Errors

If images fail to pull:
1. Verify image registry credentials are configured
2. Check that custom provisioner registry fix is applied
3. Ensure all required environment variables are exported