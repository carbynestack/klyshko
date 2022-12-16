#!/usr/bin/env bash
#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Make sure that the following versions "match", i.e., the charts application
# version matches ETCD_VERSION (see https://artifacthub.io/packages/helm/bitnami/etcd).
# Upgrading the Bitnami etcd chart does not work at the moment (see https://github.com/bitnami/bitnami-docker-etcd/pull/47).
# Hence, you have to delete the old chart manually when changing the chart
# version using
#
#     helm uninstall test-etcd
#
export ETCD_VERSION=v3.5.4
export ETCD_CHART_VERSION=8.3.1

kubectl config use-context "kind-apollo"
if ! kubectl wait --for=condition=ready pod test-etcd-0 --timeout=0s; then
  echo -e "${YELLOW}Deploying etcd${NC}"
  helm repo add bitnami https://charts.bitnami.com/bitnami --force-update
  helm upgrade test-etcd --install --wait --set auth.rbac.create=false --set service.type=LoadBalancer bitnami/etcd --version ${ETCD_CHART_VERSION}
  echo -e "${GREEN}Waiting for etcd to become available${NC}"
  if ! kubectl wait --for=condition=ready pod test-etcd-0 --timeout=120s; then
    echo -e "${RED}Failed to start etcd${NC}"
    exit 1
  fi
fi

(
cd klyshko-mp-spdz
hack/deploy.sh
)
(
cd klyshko-provisioner
hack/deploy.sh
)
(
cd klyshko-operator
hack/deploy.sh
)
