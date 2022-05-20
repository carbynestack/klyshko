# Klyshko Operator

The *Klyshko Operator* is the brain of the Klyshko subsystem. A set of
Kubernetes [Controllers] is used to drive the lifecycle of jobs (VC-level) and
tasks (VCP-level) and to decide when jobs are to be scheduled based on observing
tuple availability in [Castor].

For a high-level description of the Klyshko subsystem, its components, and how
these interact, please see the [README] at the root of this repository.

[castor]: https://github.com/carbynestack/castor
[controllers]: https://kubernetes.io/docs/concepts/architecture/controller/
[readme]: ../README.md
