#!/usr/bin/env bash
#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

declare -A typeByType=(
  ["bit_gfp"]="BIT_GFP"
  ["bit_gf2n"]="BIT_GF2N"
  ["inputmask_gfp"]="INPUT_MASK_GFP"
  ["inputmask_gf2n"]="INPUT_MASK_GF2N"
  ["inversetuple_gfp"]="INVERSE_TUPLE_GFP"
  ["inversetuple_gf2n"]="INVERSE_TUPLE_GF2N"
  ["squaretuple_gfp"]="SQUARE_TUPLE_GFP"
  ["squaretuple_gf2n"]="SQUARE_TUPLE_GF2N"
  ["multiplicationtriple_gfp"]="MULTIPLICATION_TRIPLE_GFP"
  ["multiplicationtriple_gf2n"]="MULTIPLICATION_TRIPLE_GF2N"
)

mkdir -p ~/.cs
cat <<EOF > cs-config
{
  "prime" : 0,
  "r" : 0,
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
  } ],
  "rinv" : 0
}
EOF

java -jar cs.jar --config-file cs-config castor upload-tuple -f "${KII_TUPLE_FILE}" -t "${typeByType[${KII_TUPLE_TYPE}]}" -i "${KII_JOB_ID}" 1
