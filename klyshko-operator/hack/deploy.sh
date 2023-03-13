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

# Checks whether the given Klyshko CRD exists.
function crd_exists() {
  kubectl get crds | grep "$1" && true
  return $?
}

# Delete all resources of the given Klyshko CRD (CRD to be provided without the klyshko.carbynestack.io/v1alpha1 suffix).
function delete_all() {
  CRD=$1
  if crd_exists "$CRD.klyshko.carbnyestack.io/v1alpha1"; then
    echo "Deleting CRD $CRD and associated resources"
    kubectl delete --all "$CRD"
  fi
}

# Makes etcdctl with version specified as environment variable ETCD_VERSION available in ./bin folder
function provide_etcdctl() {
  REQUIRED_VERSION=$1
  echo "Checking for etcdctl ${REQUIRED_VERSION} in $(pwd)/bin"
  if [[ -f "bin/etcdctl" ]]
  then
      INSTALLED_VERSION="v$(bin/etcdctl version | head -1 | cut -c 18-)"
      if [[ ${INSTALLED_VERSION} != "${REQUIRED_VERSION}" ]]
      then
        echo "Removing version ${INSTALLED_VERSION}"
        rm bin/etcdctl
      else
        return 0
      fi
  fi
  echo "Downloading version ${REQUIRED_VERSION}"
  DOWNLOAD_URL=https://github.com/etcd-io/etcd/releases/download
  rm -f "/tmp/etcd-${REQUIRED_VERSION}-linux-amd64.tar.gz"
  curl -L "${DOWNLOAD_URL}/${REQUIRED_VERSION}/etcd-${REQUIRED_VERSION}-linux-amd64.tar.gz" -o "/tmp/etcd-${REQUIRED_VERSION}-linux-amd64.tar.gz"
  mkdir -p bin
  tar --extract --file="/tmp/etcd-${REQUIRED_VERSION}-linux-amd64.tar.gz" -C bin/ "etcd-${REQUIRED_VERSION}-linux-amd64/etcdctl" --strip-components=1
  rm -f "/tmp/etcd-${REQUIRED_VERSION}-linux-amd64.tar.gz"
}

echo -e "${YELLOW}Deploying Klyshko Operator${NC}"

declare -a CLUSTERS=("starbuck" "apollo")

echo -e "${GREEN}Undeploying from clusters${NC}"
for c in "${CLUSTERS[@]}"
do
  echo -e "${GREEN}Undeploying from $c${NC}"
  kubectl config use-context "kind-$c"
  delete_all tuplegenerationjobs
  delete_all tuplegenerationtasks
  if ! make undeploy IMG="carbynestack/klyshko-operator:v0.0.1"; then
    echo -e "${RED}Undeploying operator failed. This is fine, if the operator has not been deployed before.${NC}"
  fi
done

echo -e "${GREEN}Cleaning up etcd${NC}"
provide_etcdctl "${ETCD_VERSION}"
bin/etcdctl --endpoints 172.18.1.129:2379 del "/klyshko/roster" --prefix

echo -e "${GREEN}Building code and image${NC}"
make docker-build IMG="carbynestack/klyshko-operator:v0.0.1"

echo -e "${GREEN}Loading docker images into cluster registries${NC}"
for c in "${CLUSTERS[@]}"
do
  echo -e "${GREEN}Loading docker image into $c${NC}"
  kind load docker-image carbynestack/klyshko-operator:v0.0.1 --name "$c"
done

for c in "${CLUSTERS[@]}"
do
  echo -e "${GREEN}Deploying in $c${NC}"
  kubectl config use-context "kind-$c"
  kubectl apply -f "config/samples/$c-vcp.yaml"
  if [ "$c" == "apollo" ]; then
    MAC_KEY_SHARE_P=$(echo "-88222337191559387830816715872691188861" | base64)
    MAC_KEY_SHARE_2=$(echo "f0cf6099e629fd0bda2de3f9515ab72b" | base64)
    EXTRA_MAC_KEY_SHARE_P="1113507028231509545156335486838233835"
    EXTRA_MAC_KEY_SHARE_2="c347ce3d9e165e4e85221f9da7591d98"
    OTHER_PLAYER_ID=1
  else
    MAC_KEY_SHARE_P=$(echo "1113507028231509545156335486838233835" | base64)
    MAC_KEY_SHARE_2=$(echo "c347ce3d9e165e4e85221f9da7591d98" | base64)
    EXTRA_MAC_KEY_SHARE_P="-88222337191559387830816715872691188861"
    EXTRA_MAC_KEY_SHARE_2="f0cf6099e629fd0bda2de3f9515ab72b"
    OTHER_PLAYER_ID=0
  fi
  sed -e "s/MAC_KEY_SHARE_P/${MAC_KEY_SHARE_P}/" -e "s/MAC_KEY_SHARE_2/${MAC_KEY_SHARE_2}/" config/samples/engine-params-secret.yaml.template > "/tmp/$c-engine-params-secret.yaml"
  sed -e "s/MAC_KEY_SHARE_P/${EXTRA_MAC_KEY_SHARE_P}/" -e "s/MAC_KEY_SHARE_2/${EXTRA_MAC_KEY_SHARE_2}/" -e "s/PLAYER_ID/${OTHER_PLAYER_ID}/" config/samples/engine-params-extra.yaml.template > "/tmp/$c-engine-params-extra.yaml"
  kubectl apply -f "/tmp/$c-engine-params-secret.yaml"
  kubectl apply -f "/tmp/$c-engine-params-extra.yaml"
  kubectl apply -f config/samples/engine-params.yaml

  make deploy IMG="carbynestack/klyshko-operator:v0.0.1"
  if [ "$c" == "apollo" ]; then
    kubectl apply -f config/samples/klyshko_v1alpha1_tuplegenerationscheduler.yaml
  fi
done
