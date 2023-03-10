#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
FROM ghcr.io/carbynestack/openjdk:jre8-20210827

ARG RELEASE_PAGE="https://github.com/carbynestack/klyshko/releases"

# Download Carbyne Stack CLI Jar at specified version
ARG CLI_VERSION=0.2-SNAPSHOT-2336890983-14-a4260ab
ADD https://github.com/carbynestack/cli/releases/download/$CLI_VERSION/cli-${CLI_VERSION}-jar-with-dependencies.jar cs.jar

# Copy resources
COPY 3RD-PARTY-LICENSES /3RD-PARTY-LICENSES
COPY kii-provisioner-run.sh .

RUN printf "\n## Klyshko Provisioner\n\
General information about third-party software components and their licenses, \
which are distributed with Klyshko Provisioner, can be found in the \
[SBOM](./sbom.json). Further details are available in the subfolder for the \
respective component or can be downloaded from the \
[Klyshko Release Page](%s).\n" "${RELEASE_PAGE}"\
  >> /3RD-PARTY-LICENSES/disclosure.md

ENTRYPOINT ["/bin/bash", "-c", "./kii-provisioner-run.sh"]
