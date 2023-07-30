#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
FROM python:3.11.3-bullseye as buildenv

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential=12.9 \
    clang-11=1:11.0.1-2 \
    libclang-dev=1:11.0-51+nmu5 \
    cmake=3.18.4-2+deb11u1 \
    libboost-dev=1.74.0.3 \
    libboost-thread-dev=1.74.0.3 \
    libntl-dev=11.4.3-1+b1 \
    libsodium-dev=1.0.18-1 \
    libssl-dev=1.1.1n-0+deb11u5 \
    gdb=10.1-1.7 \
    valgrind=1:3.16.1-1 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip==23.1.2 ipython==8.14.0

COPY MP-SPDZ /MP-SPDZ
COPY cowgear-offline.cpp /MP-SPDZ/Utils/cowgear-offline.cpp
COPY CONFIG.mine /MP-SPDZ/CONFIG.mine
WORKDIR /MP-SPDZ

RUN make clean
RUN make -j boost
RUN make -j cowgear-offline.x

FROM ghcr.io/carbynestack/ubuntu:20.04-20210827-nonroot

ARG RELEASE_PAGE="https://github.com/carbynestack/klyshko/releases"

# Copy shared libraries and CRG executable
COPY --from=buildenv \
    /MP-SPDZ/libSPDZ.so \
    /MP-SPDZ/libFHE.so \
    /MP-SPDZ/local/lib/libboost_filesystem.so.1.81.0 \
    /MP-SPDZ/local/lib/libboost_system.so.1.81.0 \
    /usr/lib/x86_64-linux-gnu/libboost_thread.so.1.74.0 \
    /usr/local/lib/
COPY --from=buildenv \
    /usr/lib/x86_64-linux-gnu/libgmpxx.so.4 \
    /usr/lib/x86_64-linux-gnu/libgmp.so.10 \
    /usr/lib/x86_64-linux-gnu/libsodium.so.23 \
    /usr/lib/x86_64-linux-gnu/libntl.so.43 \
    /usr/lib/x86_64-linux-gnu/libssl.so.1.1 \
    /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 \
    /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libgf2x.so.3.0.0 /usr/lib/x86_64-linux-gnu/libgf2x.so.3
COPY --from=buildenv /MP-SPDZ/cowgear-offline.x /usr/local/bin/

# Copy KII resources
COPY 3RD-PARTY-LICENSES /3RD-PARTY-LICENSES
COPY kii-run.sh .

# Generate disclosure document
USER root:root
RUN printf "\n## Klyshko MP-SPDZ CowGear Correlated Randomness Generator (CRG)\n\
General information about third-party software components and their licenses, \
which are distributed with Klyshko MP-SPDZ CowGear CRG, can be found in the \
[SBOM](./sbom.json). Further details are available in the subfolder for the \
respective component or can be downloaded from the \
[Klyshko Release Page](%s).\n" "${RELEASE_PAGE}"\
  >> /3RD-PARTY-LICENSES/disclosure.md

#  Group and user IDs are defined in base image (non-numerical values, i.e., cs:cs, are not working in k8s)
USER 1000:1000

ENTRYPOINT ["/bin/bash", "-c", "./kii-run.sh"]
