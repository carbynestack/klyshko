# Changelog

## [0.4.0](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.3.0...operator-chart-v0.4.0) (2023-09-04)


### ⚠ BREAKING CHANGES

* **operator-chart:** implement pod template support ([#89](https://github.com/carbynestack/klyshko/issues/89))

### Features

* **operator-chart:** implement pod template support ([#89](https://github.com/carbynestack/klyshko/issues/89)) ([e8a6e09](https://github.com/carbynestack/klyshko/commit/e8a6e0953c23739311a3240d38b84b0d683b65d4))

## [0.3.0](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.2.0...operator-chart-v0.3.0) (2023-08-03)


### ⚠ BREAKING CHANGES

* **operator/operator-chart/mp-spdz-cowgear:** add support for inter-CRG networking and CowGear CRG ([#65](https://github.com/carbynestack/klyshko/issues/65))
* **operator/operator-chart:** add tuple generator CRG and support for tuple-type-specific thresholds and priorities ([#70](https://github.com/carbynestack/klyshko/issues/70))

### Features

* **operator/operator-chart/mp-spdz-cowgear:** add support for inter-CRG networking and CowGear CRG ([#65](https://github.com/carbynestack/klyshko/issues/65)) ([e356440](https://github.com/carbynestack/klyshko/commit/e356440f8b9bd5a7452ae0b9476e101bfc6926bc))
* **operator/operator-chart:** add tuple generator CRG and support for tuple-type-specific thresholds and priorities ([#70](https://github.com/carbynestack/klyshko/issues/70)) ([e216603](https://github.com/carbynestack/klyshko/commit/e2166031ed57fd9c982f0f8ed697f3dfa4d4aabd))

## [0.2.0](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.7...operator-chart-v0.2.0) (2023-06-23)


### Features

* **operator-chart:** Add etcd endpoint configuration value ([#66](https://github.com/carbynestack/klyshko/issues/66)) ([5c065fe](https://github.com/carbynestack/klyshko/commit/5c065fe59bfded1e2d348e36ec9c46ed9eea51b4))

## [0.1.7](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.6...operator-chart-v0.1.7) (2023-04-17)


### Bug Fixes

* **operator-chart:** add property definition for generated tuples per job ([f711469](https://github.com/carbynestack/klyshko/commit/f711469ff118e4508f16657456a6b6ed60667c61))

## [0.1.6](https://github.com/carbynestack/klyshko/compare/operator-chart-v0.1.5...operator-chart-v0.1.6) (2023-03-22)


### Bug Fixes

* **mp-spdz/operator-chart:** empty commit to trigger publication of artifacts ([#30](https://github.com/carbynestack/klyshko/issues/30)) ([f9beb81](https://github.com/carbynestack/klyshko/commit/f9beb81703fe8a14f568437cd29b7362381ae402))
* **mp-spdz/operator-chart:** trigger workflows ([#37](https://github.com/carbynestack/klyshko/issues/37)) ([1a754c3](https://github.com/carbynestack/klyshko/commit/1a754c336d4cef441b1cbcaeb4820d034c38b90e))
* **mp-spdz/operator-chart:** trigger workflows ([#41](https://github.com/carbynestack/klyshko/issues/41)) ([bf8b9b0](https://github.com/carbynestack/klyshko/commit/bf8b9b0a51d85473d6bf785dfd0efab608124ccc))
* **mp-spdz:** trigger workflow ([5ab6139](https://github.com/carbynestack/klyshko/commit/5ab6139349bc6349045128edde210f7d337de47d))
* **operator-chart:** remove legacy chart folder incl. migration of changelog updates ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
* **operator-chart:** rename chart to make publication workflow work ([#47](https://github.com/carbynestack/klyshko/issues/47)) ([b529207](https://github.com/carbynestack/klyshko/commit/b5292070fda11633f8b61b972dce4882a6e7bef1))
* **operator:** enable processing of historical / missed etcd-backed roster updates ([#49](https://github.com/carbynestack/klyshko/issues/49)) ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419)), closes [#15](https://github.com/carbynestack/klyshko/issues/15)
* **operator:** get rid of unsupported trace logging level ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
* **operator:** use numeric user and group ID in operator dockerfile ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))
* **operator:** use retries to address race condition on etcd roster updates ([cf5b0f6](https://github.com/carbynestack/klyshko/commit/cf5b0f67e6a3e5ca2a6525e4b65b511a976d8419))

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
