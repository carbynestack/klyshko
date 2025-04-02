# Klyshko Operator Chart

Helm chart for the Klyshko operator. [Klyshko](../../README.md) is the
correlated randomness generation subsystem of Carbyne Stack.

## TL;DR

```bash
helm install klyshko
```

## Introduction

This chart deploys the
[Klyshko Operator](https://github.com/carbynestack/klyshko/klyshko-operator) on
a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh)
package manager.

> **Tip**: This chart is used in the `helmfile.d` based deployment specification
> available in the
> [`carbynestack/carbynestack`](https://github.com/carbynestack/carbynestack)
> repository.

## Prerequisites

- Kubernetes 1.18+ (may also work on earlier and later versions but has not been
  tested)

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
helm install --name my-release klyshko
```

Make sure that your current working directory is `<project-base-dir>/charts`.
The command deploys the Klyshko operator on the Kubernetes cluster in the
default configuration. The [configuration](#configuration) section lists the
parameters that can be configured to customize the deployment.

> **Tip**: List all releases using `helm list`

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and
deletes the release.

## Configuration

The following sections list the (main) configuration parameters of the `klyshko`
chart and their default values. For the full list of configuration parameters
see [values.yaml](values.yaml).

Specify each parameter using the `--set key=value[,key=value]` argument to
`helm install`. For example,

```bash
helm install --name my-release --set controller.image.tag=<tag> klyshko
```

The above command sets the Klyshko operator controller image version to `<tag>`.

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example,

```bash
helm install --name my-release -f values.yaml klyshko
```

### General

| Parameter          | Description                                   | Default |
| ------------------ | --------------------------------------------- | ------- |
| `imagePullSecrets` | Pull secrets used to fetch the Klyshko images | `[]`    |

### Controller

| Parameter                          | Description                                                            | Default                                              |
| ---------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------- |
| `controller.image.registry`        | Image registry used to pull the controller image                       | `ghcr.io`                                            |
| `controller.image.repository`      | Controller image name                                                  | `carbynestack/klyshko-operator-controller`           |
| `controller.image.tag`             | Controller image tag                                                   | `latest`                                             |
| `controller.image.pullPolicy`      | Controller image pull policy                                           | `IfNotPresent`                                       |
| `controller.etcd.endpoint`         | The address of the etcd service used for cross VCP coordination        | `172.18.1.129:2379`                                  |
| `controller.etcd.dialTimeout`      | The timeout (in seconds) for the etcd client to establish a connection | `5`                                                  |
| `controller.castorUrl`             | The URL of the Castor service                                          | `http://castor:8080`                                 |
| `controller.vcpIp`                 | The IP address of the VCP accessible to other VCPs in the VC           | S                                                    |
| `controller.ingress.portRange.min` | The minimum port number for the ingress port range                     | `30500`                                              |
| `controller.ingress.portRange.max` | The maximum port number for the ingress port range                     | `30504`                                              |
| `controller.egress.portRange.min`  | The minimum port number for the egress port range                      | `30500`                                              |
| `controller.egress.portRange.max`  | The maximum port number for the egress port range                      | `30550`                                              |
| `controller.egress.serviceHost`    | The hostname of the Istio egress gateway service                       | `istio-egressgateway.istio-system.svc.cluster.local` |
| `controller.egress.gatewayName`    | The name of the Istio Gateway used for egress traffic                  | `partner-egressgateway`                              |
| `controller.tls.enabled`           | Enable TLS for inter-VCP communication                                 | `false`                                              |
| `controller.tls.secretName`        | The k8s secret containing the TLS client and CA certificates           |                                                      |

### Provisioner

| Parameter                      | Description                                       | Default                            |
| ------------------------------ | ------------------------------------------------- | ---------------------------------- |
| `provisioner.image.registry`   | Image registry used to pull the provisioner image | `ghcr.io`                          |
| `provisioner.image.repository` | Provisioner image name                            | `carbynestack/klyshko-provisioner` |
| `provisioner.image.tag`        | Provisioner image tag                             | `latest`                           |
