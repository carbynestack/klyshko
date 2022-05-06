#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
FROM ghcr.io/carbynestack/openjdk:jre8-20210827

ARG CLI_VERSION=0.1-SNAPSHOT-1576571202-7-cf3db5b

RUN apt-get update && apt-get install -y --no-install-recommends \
    gettext-base \
    && rm -rf /var/lib/apt/lists/*

# Download Carbyne Stack CLI Jar at specified version
# RUN curl -o cs.jar -L https://github.com/carbynestack/cli/releases/download/$CLI_VERSION/cli-$CLI_VERSION-jar-with-dependencies.jar
COPY cs.jar .

# Copy resources
COPY kii-provisioner-run.sh .

CMD ["/bin/bash", "-c", "./kii-provisioner-run.sh"]