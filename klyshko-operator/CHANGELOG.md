# Changelog

## [0.3.0](https://github.com/carbynestack/klyshko/compare/operator-v0.2.0...operator-v0.3.0) (2023-08-03)


### ⚠ BREAKING CHANGES

* **operator:** implement pod template support ([#86](https://github.com/carbynestack/klyshko/issues/86))
* **operator/operator-chart/mp-spdz-cowgear:** add support for inter-CRG networking and CowGear CRG ([#65](https://github.com/carbynestack/klyshko/issues/65))
* **operator/operator-chart:** add tuple generator CRG and support for tuple-type-specific thresholds and priorities ([#70](https://github.com/carbynestack/klyshko/issues/70))

### Features

* **operator/operator-chart/mp-spdz-cowgear:** add support for inter-CRG networking and CowGear CRG ([#65](https://github.com/carbynestack/klyshko/issues/65)) ([e356440](https://github.com/carbynestack/klyshko/commit/e356440f8b9bd5a7452ae0b9476e101bfc6926bc))
* **operator/operator-chart:** add tuple generator CRG and support for tuple-type-specific thresholds and priorities ([#70](https://github.com/carbynestack/klyshko/issues/70)) ([e216603](https://github.com/carbynestack/klyshko/commit/e2166031ed57fd9c982f0f8ed697f3dfa4d4aabd))
* **operator:** implement pod template support ([#86](https://github.com/carbynestack/klyshko/issues/86)) ([c1ea1f4](https://github.com/carbynestack/klyshko/commit/c1ea1f47b1fab7a7220e919aa62d0acf67989670)), closes [#73](https://github.com/carbynestack/klyshko/issues/73)

## [0.2.0](https://github.com/carbynestack/klyshko/compare/operator-v0.1.1...operator-v0.2.0) (2023-03-22)


### Features

* **operator/mp-spdz:** make number of tuples per job configurable via scheduler CRD ([#56](https://github.com/carbynestack/klyshko/issues/56)) ([90eb906](https://github.com/carbynestack/klyshko/commit/90eb906c3a9540db39b6072947e686801e1de68c))

## [0.1.1](https://github.com/carbynestack/klyshko/compare/operator-v0.1.0...operator-v0.1.1) (2023-03-20)


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

## [0.1.0](https://github.com/carbynestack/klyshko/compare/operator-v0.0.1...operator-v0.1.0) (2023-03-13)


### Features

* **mp-spdz/operator/operator-chart/provisioner:** initial commit for k8s operator based implementation ([b4da582](https://github.com/carbynestack/klyshko/commit/b4da58202091eefcea3782070587f094d9dabb83))
