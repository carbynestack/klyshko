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
NC='\033[0m' # No Color

echo -e "${YELLOW}Deploying Klyshko Provisioner${NC}"

declare -a CLUSTERS=("starbuck" "apollo")

echo -e "${GREEN}Building code and image${NC}"
docker build -t carbynestack/klyshko-provisioner:1.0.0-SNAPSHOT . -f Dockerfile

echo -e "${GREEN}Loading docker images into cluster registries${NC}"
for c in "${CLUSTERS[@]}"
do
   echo -e "${GREEN}Loading docker image into $c${NC}"
   kind load docker-image carbynestack/klyshko-provisioner:1.0.0-SNAPSHOT --name "$c"
done
