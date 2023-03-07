# Carbyne Stack Klyshko Correlated Randomness Generation

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/3a07fd83b67647138b8ea660d16cdc35)](https://www.codacy.com/gh/carbynestack/klyshko/dashboard?utm_source=github.com&utm_medium=referral&utm_content=carbynestack/klyshko&utm_campaign=Badge_Grade)
[![codecov](https://codecov.io/gh/carbynestack/klyshko/branch/master/graph/badge.svg?token=6hRb7xRW6C)](https://codecov.io/gh/carbynestack/klyshko)
[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit&logoColor=white)](https://github.com/pre-commit/pre-commit)
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE_OF_CONDUCT.md)

Klyshko is a kubernetes-native open source correlated randomness generator (CRG)
service for Secure Multiparty Computation in the offline/online model and part
of [Carbyne Stack](https://github.com/carbynestack).

> **DISCLAIMER**: Carbyne Stack Klyshko is in *proof-of-concept* stage. The
> software is not ready for production use. It has neither been developed nor
> tested for a specific use case.

## Namesake

*Klyshko* is one of the inventors of *spontaneous parametric down-conversion*
(SPDC). SPDC is an important process in quantum optics, used especially as a
source of entangled photon pairs, and of single photons (see
[Wikipedia](https://en.wikipedia.org/wiki/Spontaneous_parametric_down-conversion)).
The analogy to the *Klyshko* service is that secret shared tuples are correlated
and thus kind of "entangled" and that the microservice is the implementation of
the process that creates the tuples.

## Architecture

Klyshko consists of three main components:

- *Correlated Randomness Generators (CRGs)* are the workhorses within Klyshko.
  They are actually generating correlated randomness. CRGs are packaged as
  Docker images and have to implement the
  [Klyshko Integration Interface (KII)](#klyshko-integration-interface-kii).
- The *Klyshko Operator* coordinates the invocation of CRGs across the VCPs in a
  VC. It consists of a number of components implemented as a Kubernetes API
  called `klyshko.carbnyestack.io/v1alpha1` providing the following
  [Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/):
  - A *Scheduler* (kind: `TupleGenerationScheduler`) monitors the availability
    of correlated randomness within the VC using the
    [Castor](https://github.com/carbynestack/castor) *Telemetry API* and
    schedules CRG invocations accordingly.
  - A *Job* (kind: `TupleGenerationJob`) abstracts a CRG invocation across the
    VCPs of a VC. The job holds the specification of the correlated randomness
    to be generated including tuple type and the number of tuples to be
    generated.
  - A *Task* (kind: `TupleGenerationTask`) represents a local or remote
    execution of a CRG. A task exposes the state of the invocation on a single
    VCP. On the job level, task states are aggregated into a job state. Remote
    tasks are proxied locally to make their state available to the job
    controller. The task controller makes use of the
    [Klyshko Integration Interface (KII)](#klyshko-integration-interface-kii) to
    interact with different CRG implementations in an implementation-independent
    way.
- The *Klyshko Provisioner* is used to upload the generated correlated
  randomness to [Castor](https://github.com/carbynestack/castor).

Klyshko uses an [etcd](https://etcd.io/) cluster to manage distributed state and
to orchestrate actions across VCPs.

## Usage

To deploy Klyshko to your VC you have to perform the following steps:

### Provide VCP Configuration

> **NOTE**: This is a workaround until the Carbyne Stack Operator (see
> [CSEP-0053](https://github.com/carbynestack/carbynestack/pull/54)) is
> available.

Klyshko needs to know the overall number of VCPs in the VC and the zero-based
index of the local VCP. This done by creating a configuration map with the
following content:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
name: cs-vcp-config
data:
playerCount: <<NUMBER-OF-VCPS>>
playerId: <<ZERO-BASED-INDEX-OF-LOCAL-VCP>>
```

### Install the operator

The Klyshko operator can be deployed either by building from source or by using
`helm`. Both variants are described below. Remember to perform the respective
steps on all VCPs of your VC.

#### From Source

You can use the `make` tool to build and deploy the operator using

```shell
cd klyshko-operator
make deploy IMG="carbynestack/klyshko-operator:v0.1.0"
```

#### Using Helm

You can deploy the Klyshko operator using `helm` as follows:

```shell
HELM_EXPERIMENTAL_OCI=1 helm install klyshko-operator \
  oci://ghcr.io/carbynestack/klyshko-operator \
  --version 0.1.0
```

### Provide the Configuration

Klyshko requires CRG-specific configuration that is provided via K8s config maps
and secrets (see [here](#configuration-parameters) for details). Consult the
documentation of the [MP-SPDZ CRG](klyshko-mp-spdz/README.md) for information of
what has to be provided.

### Instantiating a Scheduler

After configuration is done, you create a scheduler on **one** of the clusters
by applying the respective manifest, e.g.,

```yaml
apiVersion: klyshko.carbnyestack.io/v1alpha1
kind: TupleGenerationScheduler
metadata:
  name: sample-crg-scheduler
spec:
  concurrency: 3
  threshold: 500000
  generator:
    image: carbynestack/klyshko-mp-spdz:1.0.0-SNAPSHOT
    imagePullPolicy: IfNotPresent
```

Klyshko will start producing correlated randomness using the given CRG image by
creating jobs whenever the number of tuples for a specific type drops below the
given `threshold`. `concurrency` specifies the maximum number of jobs that are
allowed to run concurrently. This is the upper limit across jobs for all tuple
types.

## Klyshko Integration Interface (KII)

> **IMPORTANT**: This is an initial incomplete version of the KII that is
> subject to change without notice. For the time being it is very much
> influenced by the CRGs provided as part of the
> [MP-SPDZ](https://github.com/data61/MP-SPDZ) project.

*Klyshko* has been designed to allow for easy integration of different
*Correlated Randomness Generators* (CRGs). Integration is done by means of
providing a docker image containing the CRG that implements the *Klyshko
Integration Interface* (KII). The parameters required by the CRG are provided
using a mix of environment variables and files made available to the container
during execution. See below for a detailed description.

> **TIP**: For an example of how to integrate the
> [MP-SPDZ](https://github.com/data61/MP-SPDZ) CRG producing *fake* tuples see
> the [klyshko-mp-spdz](klyshko-mp-spdz) module.

### Entrypoint

The CRG docker image must spawn the tuple generation process when launched as a
container. The command given as the entrypoint must terminate with a non-zero
exit code if and only if the tuples could not be generated for some reason.

### Environment Variables

The following environment variables are passed into CRG containers to control
the tuple generation and provisioning process.

#### Input

- `KII_JOB_ID`: The Type 4 UUID used as a job identifier. This is the same among
  all VCPs in the VC.
- `KII_TUPLES_PER_JOB`: The number of tuples to be generated. The CRG should
  make its best effort to match the requested number but is not required to do
  so in case optimizations like batching mandate it.
- `KII_PLAYER_NUMBER`: The 0-based number of the local VCP.
- `KII_PLAYER_COUNT`: The overall number of VCPs in the VC.
- `KII_TUPLE_TYPE`: The tuple type to generate. Must be one of
  - `BIT_GFP`, `BIT_GF2N`
  - `INPUT_MASK_GFP`, `INPUT_MASK_GF2N`
  - `INVERSE_TUPLE_GFP`, `INVERSE_TUPLE_GF2N`
  - `SQUARE_TUPLE_GFP`, `SQUARE_TUPLE_GF2N`
  - `MULTIPLICATION_TRIPLE_GFP`, `MULTIPLICATION_TRIPLE_GF2N`

#### Output

- `KII_TUPLE_FILE`: The file the generated tuples must be written to.

### Configuration Parameters

CRGs typically require some configuration that has to be provided using K8s
config maps and secrets. While the existence of these resources is dictated by
the KII, their content is CRG implementation specific. Please refer to the CRG
documentation for detailed information on what is expected. The following
examples are for the [MP-SPDZ CRG](klyshko-mp-spdz/README.md).

#### Public Parameters

Public, i.e., non-sensitive, parameters are provided in a config map with name
`io.carbynestack.engine.params` as follows:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: io.carbynestack.engine.params
data:
  prime: <<PRIME>>
```

They are provided to CRGs as files in the `/etc/kii/params/` folder. The file
`/etc/kii/params/prime` contains the `<<PRIME>>` in the example above.

#### Secret Parameters

Sensitive parameters are provided using a K8s secret with name
`io.carbynestack.engine.params.secret` as follows:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: io.carbynestack.engine.params.secret
type: Opaque
data:
  mac_key_share_p: |
    <<MAC_KEY_SHARE_P>>
  mac_key_share_2: |
    <<MAC_KEY_SHARE_2>>
```

They are made available to CRGs as files in the folder `/etc/kii/secret-params`.

#### Additional Parameters

Additional parameters *may* be provided using a K8s config map with name
`io.carbynestack.engine.params.extra` as follows:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: io.carbynestack.engine.params.extra
data:
  <<KEY-#1>>: <<VALUE-#1>>
  <<KEY-#2>>: <<VALUE-#2>>
```

These are made available to CRGs by the Klyshko runtime as files in folder
`/etc/kii/extra-params`. For an example of how this is used see the
[MP-SPDZ fake tuple CRG][mp-spdz-fake].

## Development

The `deploy.sh` scripts in the `hack` folders (top-level and within modules) can
be used to (re-)deploy Klyshko to a 2-party Carbyne Stack VC setup as described
in the [tutorials](https://carbynestack.io/getting-started) on the Carbyne Stack
website. To trigger (re-)deployment, the top-level script must be called from
the project root folder using

```shell
./hack/deploy.sh
```

### Logging

#### Verbosity

The Klyshko operator uses the [logging infrastructure][o-sdk-logging] provided
by the Operator SDK. To adjust the logging verbosity set the `zap-log-level`
flag to either `info`, `error`, or any integer value > 0 (higher values = more
verbose, see table below).

```yaml
apiVersion: apps/v1
kind: Deployment
# ...
spec:
  template:
    spec:
      containers:
        # ...
        - name: manager
          args:
            # ...
            - "--zap-log-level=<<LOG-LEVEL>>"
```

#### Choosing Log Levels

We use the following logging level convention in the Klyshko code basis.

| Meaning   | Level | Command                          |
| --------- | ----- | -------------------------------- |
| Essential | 0     | `logger.Info()/Error()`          |
| Debug     | 5     | `logger.V(DEBUG).Info()/Error()` |
| Tracing   | 10    | `logger.V(TRACE).Info()/Error()` |

## License

Carbyne Stack *Klyshko Correlated Randomness Generation Service* is open-sourced
under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

### 3rd Party Licenses

For information on how license obligations for 3rd party OSS dependencies are
fulfilled see the [README](https://github.com/carbynestack/carbynestack) file of
the Carbyne Stack repository.

## Contributing

Please see the Carbyne Stack
[Contributor's Guide](https://github.com/carbynestack/carbynestack/blob/master/CONTRIBUTING.md)
.

[mp-spdz-fake]: klyshko-mp-spdz/README.md#additional-parameters
[o-sdk-logging]: https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/
