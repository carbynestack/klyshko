#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
FROM python:3.11.3-bullseye as buildenv

RUN apt-get update && apt-get install -y --no-install-recommends \
    automake \
    build-essential \
    clang-11 \
    cmake \
    git \
    libboost-dev \
    libboost-thread-dev \
    libclang-dev \
    libgmp-dev \
    libntl-dev \
    libsodium-dev \
    libssl-dev \
    libtool \
    vim \
    gdb \
    valgrind \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --upgrade pip ipython

COPY MP-SPDZ /MP-SPDZ
COPY cowgear-offline.cpp /MP-SPDZ/Utils/cowgear-offline.cpp
COPY CONFIG.mine /MP-SPDZ/CONFIG.mine
WORKDIR /MP-SPDZ

RUN make clean
RUN make -j boost
RUN make -j cowgear-offline.x

# TODO Replace with compliant base image
FROM ubuntu:22.04

COPY --from=buildenv /MP-SPDZ/libSPDZ.so /usr/local/lib/
COPY --from=buildenv /MP-SPDZ/libFHE.so /usr/local/lib/
COPY --from=buildenv /MP-SPDZ/local/lib/libboost_filesystem.so.1.81.0 /usr/local/lib/
COPY --from=buildenv /MP-SPDZ/local/lib/libboost_system.so.1.81.0 /usr/local/lib/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libboost_thread.so.1.74.0 /usr/local/lib/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libgmpxx.so.4 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libgmp.so.10 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libsodium.so.23 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libntl.so.43 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libssl.so.1.1 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libcrypto.so.1.1 /usr/lib/x86_64-linux-gnu/
COPY --from=buildenv /usr/lib/x86_64-linux-gnu/libgf2x.so.3.0.0 /usr/lib/x86_64-linux-gnu/libgf2x.so.3
COPY --from=buildenv /MP-SPDZ/cowgear-offline.x /usr/local/bin/
