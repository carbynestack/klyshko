#!/usr/bin/env bash
#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
GREEN='\033[0;32m'
NC='\033[0m' # No Color

declare -a CLUSTERS=("starbuck" "apollo")

echo -e "${GREEN}Undeploying from clusters${NC}"
for c in "${CLUSTERS[@]}"
do
  echo -e "${GREEN}Undeploying from $c${NC}"
  kubectl config use-context "kind-$c"
  if [ "$c" == "apollo" ]; then
    kubectl delete -f config/samples/klyshko_v1alpha1_tuplegenerationjob.yaml
  fi
  kubectl delete --all tgj
  kubectl delete --all tgt
  make undeploy IMG="carbynestack/klyshko-operator:v0.0.1"
done

echo -e "${GREEN}Cleaning up etcd${NC}"
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
  make deploy IMG="carbynestack/klyshko-operator:v0.0.1"
  if [ "$c" == "apollo" ]; then
    kubectl apply -f config/samples/klyshko_v1alpha1_tuplegenerationjob.yaml
  fi
done