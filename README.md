# Carbyne Stack Klyshko Correlated Randomness Generation

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/3a07fd83b67647138b8ea660d16cdc35)](https://www.codacy.com/gh/carbynestack/klyshko/dashboard?utm_source=github.com&utm_medium=referral&utm_content=carbynestack/klyshko&utm_campaign=Badge_Grade)
[![codecov](https://codecov.io/gh/carbynestack/klyshko/branch/master/graph/badge.svg?token=6hRb7xRW6C)](https://codecov.io/gh/carbynestack/klyshko)
[![Conventional Commits](https://img.shields.io/badge/Conventional%20Commits-1.0.0-%23FE5196?logo=conventionalcommits&logoColor=white)](https://conventionalcommits.org)
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

- *Correlated Randomness Generators (CRGs)* (kind: `TupleGenerator`) are the
  workhorses within Klyshko. They are actually generating correlated randomness.
  CRGs are packaged as Docker images and have to implement the
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
make deploy IMG="carbynestack/klyshko-operator:v0.3.0"
```

#### Using Helm

You can deploy the Klyshko operator using `helm` as follows:

```shell
HELM_EXPERIMENTAL_OCI=1 helm install klyshko \
  oci://ghcr.io/carbynestack/klyshko \
  --version 0.3.0
```

### Provide the Configuration

Klyshko requires CRG-specific configuration that is provided via K8s config maps
and secrets (see [here](#configuration-parameters) for details). Consult the
documentation of the [MP-SPDZ CRG](klyshko-mp-spdz/README.md) for information of
what has to be provided.

### Registering a Tuple Generator

After configuration is done, you can create a Tuple Generator using, e.g.,

```shell
cat <<EOF | kubectl apply -f -
apiVersion: klyshko.carbnyestack.io/v1alpha1
kind: TupleGenerator
metadata:
  name: mp-spdz-fake
spec:
  template:
    spec:
      container:
        image: carbynestack/klyshko-mp-spdz:0.2.0
  supports:
    - type: BIT_GFP
      batchSize: 100000
    - type: INPUT_MASK_GFP
      batchSize: 100000
    - type: INVERSE_TUPLE_GFP
      batchSize: 100000
    - type: SQUARE_TUPLE_GFP
      batchSize: 100000
    - type: MULTIPLICATION_TRIPLE_GFP
      batchSize: 100000
    - type: BIT_GF2N
      batchSize: 100000
    - type: INPUT_MASK_GF2N
      batchSize: 100000
    - type: INVERSE_TUPLE_GF2N
      batchSize: 100000
    - type: SQUARE_TUPLE_GF2N
      batchSize: 100000
    - type: MULTIPLICATION_TRIPLE_GF2N
      batchSize: 100000
EOF
```

This registers the generator with Klyshko. Note that you have to specify each
tuple type supported by the CRG and provide a recommended batch size for jobs
that generate that type of tuples. Please consult the CRG documentation for more
information.

> **IMPORTANT**: In case a tuple type is supported by multiple generators no
> tuples are generated for that tuple type to avoid potential inconsistencies
> across VCPs.

#### CRG Pod Template

You can customize some aspects of the pod launched for a tuple generation task,
i.e., the pod that hosts the container running the generator image. The
following fields are customizable (in lexical order):

| Aspect    | Description                                                                                        | Field(s)                                               |
| --------- | -------------------------------------------------------------------------------------------------- | ------------------------------------------------------ |
| Affinity  | Used to constrain on which nodes the generator pod can run (see [here][k8s-affinity] for details). | `spec.template.spec.affinity`                          |
| Image     | The generator image to use (see [here][k8s-images] for details).                                   | `spec.template.spec.container.{image,imagePullPolicy}` |
| Resources | How much resources the container needs (see [here][k8s-resource] for details).                     | `spec.template.spec.container.resources`               |

Note that `spec.template.spec.container.image` is the only mandatory field. If a
field is not provided the general default values for pods / containers are used
(see links provided above).

A fully customized sample generator pod template looks like the following:

```yaml
apiVersion: klyshko.carbnyestack.io/v1alpha1
kind: TupleGenerator
metadata:
  name: mp-spdz-fake
spec:
  template:
    affinity: # Only place pod on nodes running a Linux OS
      nodeAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
          nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/os
                operator: In
                values:
                  - linux
    spec:
      container:
        image: carbynestack/klyshko-mp-spdz:0.2.0
        imagePullPolicy: Always
        resources:
          requests: # Asking for 2 GB of memory and 1 CPU unit (physical or virtual CPU core)
            memory: "2G"
            cpu: "1"
  supports:
    - ...
```

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
  policies:
    - type: BIT_GFP
      threshold: 1000000
    - type: INPUT_MASK_GFP
      threshold: 1000000
      priority: 10
    - type: INVERSE_TUPLE_GFP
      threshold: 1000000
    - type: SQUARE_TUPLE_GFP
      threshold: 1000000
    - type: MULTIPLICATION_TRIPLE_GFP
      threshold: 1000000
      priority: 10
    - type: BIT_GF2N
      threshold: 100000
    - type: INPUT_MASK_GF2N
      threshold: 100000
    - type: INVERSE_TUPLE_GF2N
      threshold: 100000
    - type: SQUARE_TUPLE_GF2N
      threshold: 100000
    - type: MULTIPLICATION_TRIPLE_GF2N
      threshold: 100000
```

Klyshko will start producing correlated randomness for the given tuple types
according to the respective *policy*. Klyshko will run CRGs in parallel as
specified by the `concurrency` parameter. If the number of running jobs drops
below that number, Klyshko selects the next tuple type to launch a job for using
a lottery scheduler. The number of tickets assigned to a tuple type is specified
by the optional `priority` parameter (default is `1` when not given). Only those
tuple types for which less than `threshold` number of tuples are available in
Castor are eligible for scheduling.

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

### SDK-based

The recommended and future-proof way to deploy Klyshko during development is by
means of the Carbynestack SDK. You can customize the deployment logic to use
your local Klyshko chart and images as follows:

1. Build the Klyshko docker images locally using

   ```shell
   export VERSION=local-dev

   # Build the MP-SPDZ fake CRG image
   pushd klyshko-mp-spdz
   docker build -f Dockerfile.fake-offline . -t "ghcr.io/carbynestack/klyshko-mp-spdz:${VERSION}"
   popd

   # and the MP-SDPZ CowGear CRG image
   pushd klyshko-mp-spdz-cowgear
   docker build -f Dockerfile . -t "ghcr.io/carbynestack/klyshko-mp-spdz-cowgear:${VERSION}"
   popd

   # Build the Provisioner image
   pushd klyshko-provisioner
   docker build -f Dockerfile . -t "ghcr.io/carbynestack/klyshko-provisioner:${VERSION}"
   popd

   # Build the Operator image
   pushd klyshko-operator
   make docker-build IMG="ghcr.io/carbynestack/klyshko-operator:${VERSION}"
   popd
   ```

1. Make the SDK locally available by cloning the
   [carbynestack/carbynestack](https://github.com/carbynestack/carbynestack)
   repository

   ```shell
   git clone git@github.com:carbynestack/carbynestack.git sdk
   cd sdk
   ```

1. Load the generated docker images into your kind clusters using

   ```shell
   declare -a CLUSTERS=("starbuck" "apollo")
   for c in "${CLUSTERS[@]}"
   do
     kind load docker-image \
       "ghcr.io/carbynestack/klyshko-mp-spdz:${VERSION}" \
       "ghcr.io/carbynestack/klyshko-mp-spdz-cowgear:${VERSION}" \
       "ghcr.io/carbynestack/klyshko-provisioner:${VERSION}" \
       "ghcr.io/carbynestack/klyshko-operator:${VERSION}" \
       --name "$c"
   done
   ```

1. Update the Klyhsko helm chart in the SDK to use your local chart by
   substituting the line

   ```shell
   ...
   chart: carbynestack-oci/klyshko
   ...
   ```

   with

   ```shell
   ...
   chart: <YOUR_KLYSHKO_REPOSITORY_ROOT>/klyshko-operator/charts/klyshko
   ...
   ```

   in the file `<YOUR_SDK_REPOSITORY_ROOT>/helmfile.d/0400.klyshko.yaml`.

1. Deploy Carbyne Stack with your locally build artifacts via

   ```shell
   # Overwrite Klyshko images used for deployment
   export KLYSHKO_GENERATOR_IMAGE_TAG="${VERSION}"
   export KLYSHKO_PROVISIONER_IMAGE_TAG="${VERSION}"
   export KLYSHKO_OPERATOR_IMAGE_TAG="${VERSION}"

   # Configure deployment
   export APOLLO_FQDN="172.18.1.128.sslip.io"
   export STARBUCK_FQDN="172.18.2.128.sslip.io"
   export RELEASE_NAME=cs
   export DISCOVERY_MASTER_HOST=$APOLLO_FQDN
   export NO_SSL_VALIDATION=true

   # Deploy Starbuck
   export FRONTEND_URL=$STARBUCK_FQDN
   export IS_MASTER=false
   export AMPHORA_VC_PARTNER_URI=http://$APOLLO_FQDN/amphora
   kubectl config use-context kind-starbuck
   helmfile apply

   # Deploy Apollo
   export FRONTEND_URL=$APOLLO_FQDN
   export IS_MASTER=true
   export AMPHORA_VC_PARTNER_URI=http://$STARBUCK_FQDN/amphora
   export CASTOR_SLAVE_URI=http://$STARBUCK_FQDN/castor
   kubectl config use-context kind-apollo
   helmfile apply
   ```

### Script-based

> **WARNING**: This method of deploying Klyshko is deprecated. The respective
> scripts will be removed soon.

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

[k8s-affinity]: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
[k8s-images]: https://kubernetes.io/docs/concepts/containers/images/
[k8s-resource]: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
[mp-spdz-fake]: klyshko-mp-spdz/README.md#additional-parameters
[o-sdk-logging]: https://sdk.operatorframework.io/docs/building-operators/golang/references/logging/
