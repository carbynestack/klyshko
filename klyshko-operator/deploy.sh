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
  MAC_KEY_SHARE_P=$([ "$c" == "apollo" ] && echo "-88222337191559387830816715872691188861" | base64 || echo "1113507028231509545156335486838233835" | base64)
  MAC_KEY_SHARE_2=$([ "$c" == "apollo" ] && echo "f0cf6099e629fd0bda2de3f9515ab72b" | base64 || echo "c347ce3d9e165e4e85221f9da7591d98" | base64)
  sed -e "s/MAC_KEY_SHARE_P/${MAC_KEY_SHARE_P}/" -e "s/MAC_KEY_SHARE_2/${MAC_KEY_SHARE_2}/" config/samples/engine-params-secret.yaml.template > "/tmp/$c-engine-params-secret.yaml"
  EXTRA_MAC_KEY_SHARE_P=$([ "$c" == "starbuck" ] && echo "-88222337191559387830816715872691188861" || echo "1113507028231509545156335486838233835")
  EXTRA_MAC_KEY_SHARE_2=$([ "$c" == "starbuck" ] && echo "f0cf6099e629fd0bda2de3f9515ab72b" || echo "c347ce3d9e165e4e85221f9da7591d98")
  sed -e "s/MAC_KEY_SHARE_P/${EXTRA_MAC_KEY_SHARE_P}/" -e "s/MAC_KEY_SHARE_2/${EXTRA_MAC_KEY_SHARE_2}/" config/samples/engine-params-extra.yaml.template > "/tmp/$c-engine-params-extra.yaml"
  kubectl apply -f /tmp/$c-engine-params-secret.yaml
  kubectl apply -f /tmp/$c-engine-params-extra.yaml
  kubectl apply -f config/samples/engine-params.yaml

  make deploy IMG="carbynestack/klyshko-operator:v0.0.1"
  if [ "$c" == "apollo" ]; then
    kubectl apply -f config/samples/klyshko_v1alpha1_tuplegenerationjob.yaml
  fi
done