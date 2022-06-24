#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
FROM ghcr.io/carbynestack/openjdk:jre8-20210827

# Download Carbyne Stack CLI Jar at specified version
ARG CLI_VERSION=0.2-SNAPSHOT-2336890983-14-a4260ab
ADD https://github.com/carbynestack/cli/releases/download/$CLI_VERSION/cli-$CLI_VERSION-jar-with-dependencies.jar cs.jar

# Copy resources
COPY kii-provisioner-run.sh .

CMD ["/bin/bash", "-c", "./kii-provisioner-run.sh"]
