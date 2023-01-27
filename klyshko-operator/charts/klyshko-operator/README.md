# Klyshko Operator

Helm chart for the Klyshko operator. [Klyshko](../../README.md) is the
correlated randomness generation subsystem of Carbyne Stack.

## TL;DR

```bash
helm install klyshko-operator
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
helm install --name my-release klyshko-operator
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

The following sections list the (main) configurable parameters of the
`klyshko-operator` chart and their default values. For the full list of
configuration parameters see [values.yaml](values.yaml).

Specify each parameter using the `--set key=value[,key=value]` argument to
`helm install`. For example,

```bash
helm install --name my-release --set controller.image.tag=<tag> klyshko-operator
```

The above command sets the Klyshko operator controller image version to `<tag>`.

Alternatively, a YAML file that specifies the values for the parameters can be
provided while installing the chart. For example,

```bash
helm install --name my-release -f values.yaml klyshko-operator
```

### General

| Parameter          | Description                                   | Default |
| ------------------ | --------------------------------------------- | ------- |
| `imagePullSecrets` | Pull secrets used to fetch the Klyshko images | `[]`    |

### Controller

| Parameter                     | Description                                      | Default                                    |
| ----------------------------- | ------------------------------------------------ | ------------------------------------------ |
| `controller.image.registry`   | Image registry used to pull the controller image | `ghcr.io`                                  |
| `controller.image.repository` | Controller image name                            | `carbynestack/klyshko-operator-controller` |
| `controller.image.tag`        | Controller image tag                             | `latest`                                   |
| `controller.image.pullPolicy` | Controller image pull policy                     | `IfNotPresent`                             |

### Provisioner

| Parameter                      | Description                                       | Default                            |
| ------------------------------ | ------------------------------------------------- | ---------------------------------- |
| `provisioner.image.registry`   | Image registry used to pull the provisioner image | `ghcr.io`                          |
| `provisioner.image.repository` | Provisioner image name                            | `carbynestack/klyshko-provisioner` |
| `provisioner.image.tag`        | Provisioner image tag                             | `latest`                           |
