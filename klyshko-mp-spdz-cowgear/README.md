# Klyshko MP-SPDZ CowGear Correlated Randomness Generator

Provides a Klyshko *Correlated Randomness Generator* (CRG) that uses the
*CowGear* covert, dishonest majority MP-SPDZ offline phase implementation.

For more information on CowGear see:

> Marcel Keller, Valerio Pastro, and Dragos Rotaru \
> *Overdrive: Making SPDZ
> Great Again* \
> Cryptology ePrint Archive, Paper 2017/1230 \
> Available at:
> <https://eprint.iacr.org/2017/1230>

Note that we run the setup phase on each invocation of the CRG as the overhead
is low. This will probably change in the future, when we have sorted out how to
provide CRG-specific artifacts as part of the deployment process (see
[CSEP-0053]).

For a high-level description of the Klyshko subsystem, its components, and how
these interact, please see the [README] at the root of this repository.

## Configuration

The MP-SPDZ CowGear CRG expects the following configuration resources.

### Public Parameters

The only required public configuration parameter is the prime (`<<PRIME>>`
placeholder below) used for generating tuples for the prime field arithmetic. It
is specified using a config map as follows:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: io.carbynestack.engine.params
data:
  prime: <<PRIME>>
```

### Secret Parameters

The required secret parameters are the MAC key shares for the prime field
(`<<MAC_KEY_SHARE_P>>` placeholder below) and for the field of characteristic 2
(`<<MAC_KEY_SHARE_2>>` below). They are specified using a K8s secret as follows:

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

## Deployment

The CowGear CRG can be deployed by creating the following `TupleGenerator`
custom resource:

```yaml
apiVersion: klyshko.carbnyestack.io/v1alpha1
kind: TupleGenerator
metadata:
  name: mp-spdz-cowgear-crg
spec:
  image: carbynestack/klyshko-mp-spdz-cowgear:latest
  imagePullPolicy: IfNotPresent
  supports:
    - type: BIT_GFP
      batchSize: TBD
    - type: INPUT_MASK_GFP
      batchSize: TBD
    - type: INVERSE_TUPLE_GFP
      batchSize: TBD
    - type: SQUARE_TUPLE_GFP
      batchSize: 2000000
    - type: MULTIPLICATION_TRIPLE_GFP
      batchSize: TBD
    - type: BIT_GF2N
      batchSize: TBD
    - type: INPUT_MASK_GF2N
      batchSize: TBD
    - type: INVERSE_TUPLE_GF2N
      batchSize: TBD
    - type: SQUARE_TUPLE_GF2N
      batchSize: 15000
    - type: MULTIPLICATION_TRIPLE_GF2N
      batchSize: 15000
```

The batch sizes are selected in a way such that the runtime of a job is roughly
five minutes on a single core of an Intel Xeon E-2276G CPU running at 3.80GHz.

## Development

### Initializing Submodules

The MP-SPDZ codebase is made available as a git submodule. In case you forgot
the `--recurse-submodules` flag when cloning this repository, you can fix this
by running

```shell
git submodule update --init --recursive
```

### Formatting

The shell scripts provided as part of this repository *strive* to adhere to the
formatting rules of the [Google Shell Style Guide][google-ssg].

### Building from Source

You have to patch the MP-SPDZ codebase available in the `MP-SPDZ` folder with
the source code for the CowGear executable `cowgear-offline.cpp`, the MP-SPDZ
build configuration in `CONFIG.mine`, and additional make rules in
`Makefile.rules`. This is done by invoking

```shell
make init
```

You might have to install some dependencies to make compilation work (see the
[MP-SPDZ repository][mp-spdz] for details). For Ubuntu 20 the following might be
sufficient:

```shell
apt-get install -y --no-install-recommends \
  automake \
  build-essential \
  clang-11 \
  cmake \
  git \
  libboost-dev \
  libboost-thread-dev \
  libclang-dev \
  libgmp-dev \
  libntl-dev \
  libsodium-dev \
  libssl-dev \
  libtool \
  vim \
  gdb \
  valgrind

pip install --upgrade pip ipython
```

After that is done, you can build the CowGear executable using

```shell
cd MP-SPDZ
make -j boost
make -j cowgear-offline.x
```

### Building the Docker Image

You can build the docker image as follows:

```shell
docker build -f Dockerfile . -t klyshko-mp-spdz-cowgear:latest
```

### Standalone Execution of the Offline Phase

The `hack/test/direct` folder contains a configurable script to run the CRG
locally (independently of Klyshko) in an N-party setup.

### Running the Tests

To run the tests to check whether the CRG is working as expected, invoke:

```shell
cd test
bats/bin/bats roundtrip.bats
```

[csep-0053]: https://github.com/carbynestack/carbynestack/pull/54
[google-ssg]: https://google.github.io/styleguide/shellguide.html
[mp-spdz]: https://github.com/data61/MP-SPDZ
[readme]: ../README.md
