# Changelog

## [0.1.5](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.4...operator-chart-v0.1.5) (2023-03-20)

### Bug Fixes

- **operator-chart:** remove legacy chart folder incl. migration of changelog
  updates
  ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
- **operator:** enable processing of historical / missed etcd-backed roster
  updates ([#49](https://github.com/carbynestack/klyshko/issues/49))
  ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419)),
  closes [#15](https://github.com/carbynestack/klyshko/issues/15)
- **operator:** get rid of unsupported trace logging level
  ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
- **operator:** use numeric user and group ID in operator dockerfile
  ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
- **operator:** use retries to address race condition on etcd roster updates
  ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))

## [0.1.4](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.3...operator-chart-v0.1.4) (2023-03-15)

### Bug Fixes

- **mp-spdz:** trigger workflow
  ([5ab6139](https://github.com/carbynestack/klyshko/commit/5ab6139349bc6349045128edde210f7d337de47d))
- **operator-chart:** rename chart to make publication workflow work
  ([#47](https://github.com/carbynestack/klyshko/issues/47))
  ([b529207](https://github.com/carbynestack/klyshko/commit/b5292070fda11633f8b61b972dce4882a6e7bef1))

## [0.1.3](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.2...operator-chart-v0.1.3) (2023-03-15)

### Bug Fixes

- **mp-spdz/operator-chart:** trigger workflows
  ([#41](https://github.com/carbynestack/klyshko/issues/41))
  ([bf8b9b0](https://github.com/carbynestack/klyshko/commit/bf8b9b0a51d85473d6bf785dfd0efab608124ccc))

## [0.1.2](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.1...operator-chart-v0.1.2) (2023-03-15)

### Bug Fixes

- **mp-spdz/operator-chart:** trigger workflows
  ([#37](https://github.com/carbynestack/klyshko/issues/37))
  ([1a754c3](https://github.com/carbynestack/klyshko/commit/1a754c336d4cef441b1cbcaeb4820d034c38b90e))

## [0.1.1](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.0...operator-chart-v0.1.1) (2023-03-14)

### Bug Fixes

- **mp-spdz/operator-chart:** empty commit to trigger publication of artifacts
  ([#30](https://github.com/carbynestack/klyshko/issues/30))
  ([f9beb81](https://github.com/carbynestack/klyshko/commit/f9beb81703fe8a14f568437cd29b7362381ae402))

## [0.1.0](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.0.1...operator-chart-v0.1.0) (2023-03-13)

### Features

- **mp-spdz/operator/operator-chart/provisioner:** initial commit for k8s
  operator based implementation
  ([b4da582](https://github.com/carbynestack/klyshko/commit/b4da58202091eefcea3782070587f094d9dabb83))
