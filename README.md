# Carbyne Stack Klyshko Correlated Randomness Generation

Klyshko is a kubernetes-native open source correlated randomness generator
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

## Klyshko Integration Interface

> **IMPORTANT**: This is an initial incomplete version of the KII that is
> subject to change without notice. For the time being it is very much
> influenced by the CRGs provided as part of the
> [MP-SPDZ](https://github.com/data61/MP-SPDZ) project.

*Klyshko* has been designed to allow for easy integration of different
correlated randomness generators (CRGs). Integration is done by means of
providing a docker image containing the CRG that implements the *Klyshko
Integration Interface* (KII).

> **TIP**: For an example of how to integrate the
> [MP-SPDZ](https://github.com/data61/MP-SPDZ) CRG producing *fake* tuples see
> the [klyshko-mp-spdz](klyshko-mp-spdz) module.

### Entrypoint

The CRG docker image must contain a `kii-run.sh` script in the working directory
that performs the tuple generation process. The script must terminate with a
non-zero exit code in case the tuples can not be generated.

### Environment Variables

The following environment variables are passed into CRG containers to control
the tuple generation and provisioning process.

#### Input

- `KII_JOB_ID`: The Type 4 UUID used as a job identifier. This is the same among
  all VCPs in the VC.
- `KII_TUPLES_PER_JOB`: The number of tuples to be generated. The CRG should try
  to match the requested number but is not required to do so.
- `KII_PLAYER_NUMBER`: The 0-based number of the local VCP.
- `KII_PLAYER_COUNT`: The overall number of VCPs in the VC.
- `KII_TUPLE_TYPE`: The tuple type to generate. Must be one of
  - `bit_gfp`, `bit_gf2n`
  - `inputmask_gfp`, `inputmask_gf2n`
  - `inversetuple_gfp`, `inversetuple_gf2n`
  - `squaretuple_gfp`, `squaretuple_gf2n`
  - `multiplicationtriple_gfp`, `multiplicationtriple_gf2n`

#### Output

- `KII_TUPLE_FILE`: The file the generated tuples must be written to.

### Prime

The prime to be used for generating prime field tuples is provided in the file
`/etc/kii/params/prime`.

### MAC Key Shares

The MAC key shares for prime and binary fields are made available as files
`mac_key_share_p` and `mac_key_share_2` in folder `/etc/kii/secret-params`.

## License

Carbyne Stack *Klyshko Correlated Randomness Generation Subsystem* is
open-sourced under the Apache License 2.0. See the [LICENSE](LICENSE) file for
details.

### 3rd Party Licenses

For information on how license obligations for 3rd party OSS dependencies are
fulfilled see the [README](https://github.com/carbynestack/carbynestack) file of
the Carbyne Stack repository.

## Contributing

Please see the Carbyne Stack
[Contributor's Guide](https://github.com/carbynestack/carbynestack/blob/master/CONTRIBUTING.md)
.
