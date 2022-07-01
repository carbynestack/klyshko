#!/usr/bin/env bash
#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

mkdir -p ~/.cs
cat <<EOF > cs-config
{
  "prime" : 0,
  "r" : 0,
  "rinv" : 0,
  "noSslValidation" : true,
  "trustedCertificates" : [ ],
  "providers" : [ {
    "amphoraServiceUrl" : "http://ignore",
    "castorServiceUrl" : "http://cs-castor:10100",
    "ephemeralServiceUrl" : "http://ignore",
    "id" : 1,
    "baseUrl" : "http://ignore"
  }, {
    "amphoraServiceUrl" : "http://ignore",
    "castorServiceUrl" : "http://ignore",
    "ephemeralServiceUrl" : "http://ignore",
    "id" : 2,
    "baseUrl" : "http://ignore"
  } ]
}
EOF

java -jar cs.jar --config-file cs-config castor upload-tuple -f "${KII_TUPLE_FILE}" -t "${KII_TUPLE_TYPE}" -i "${KII_JOB_ID}" 1
