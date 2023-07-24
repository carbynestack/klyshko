#!/usr/bin/env bash
#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

#######################################
# Retry a command up to a specific number of times until it exits successfully,
# with exponential back off.
#
# Copied from https://gist.github.com/sj26/88e1c6584397bb7c13bd11108a579746
# published under the "The Unlicense".
#
# Globals:
#   None
# Arguments:
#   The number of retries before giving up.
#   The command to execute, including any number of arguments.
# Outputs:
#   None
# Returns:
#   0 if command was executed successfully, status code of final failing
#   command execution otherwise.
#######################################
function retry {
  local retries=$1
  shift
  local count=0
  until "$@"; do
    exit=$?
    wait=$((2 ** count))
    count=$((count + 1))
    if [ $count -lt "$retries" ]; then
      echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}

# Setup offline executable command line arguments dictionary
prime=$(cat /etc/kii/params/prime)
declare -A argsByType=(
  ["BIT_GFP"]="--field-type gfp --tuple-type bits --prime ${prime}"
  ["BIT_GF2N"]="--field-type gf2n --tuple-type bits"
  ["INPUT_MASK_GFP"]="--field-type gfp --tuple-type triples --prime ${prime}"
  ["INPUT_MASK_GF2N"]="--field-type gf2n --tuple-type triples"
  ["INVERSE_TUPLE_GFP"]="--field-type gfp --tuple-type inverses --prime ${prime}"
  ["INVERSE_TUPLE_GF2N"]="--field-type gf2n --tuple-type inverses"
  ["SQUARE_TUPLE_GFP"]="--field-type gfp --tuple-type squares --prime ${prime}"
  ["SQUARE_TUPLE_GF2N"]="--field-type gf2n --tuple-type squares"
  ["MULTIPLICATION_TRIPLE_GFP"]="--field-type gfp --tuple-type triples --prime ${prime}"
  ["MULTIPLICATION_TRIPLE_GF2N"]="--field-type gf2n --tuple-type triples"
)
pn=${KII_PLAYER_NUMBER}
pc=${KII_PLAYER_COUNT}
declare -A tupleFileByType=(
  ["BIT_GFP"]="${pc}-p-128/Bits-p-P${pn}"
  ["BIT_GF2N"]="${pc}-2-40/Bits-2-P${pn}"
  ["INPUT_MASK_GFP"]="${pc}-p-128/Triples-p-P${pn}"
  ["INPUT_MASK_GF2N"]="${pc}-2-40/Triples-2-P${pn}"
  ["INVERSE_TUPLE_GFP"]="${pc}-p-128/Inverses-p-P${pn}"
  ["INVERSE_TUPLE_GF2N"]="${pc}-2-40/Inverses-2-P${pn}"
  ["SQUARE_TUPLE_GFP"]="${pc}-p-128/Squares-p-P${pn}"
  ["SQUARE_TUPLE_GF2N"]="${pc}-2-40/Squares-2-P${pn}"
  ["MULTIPLICATION_TRIPLE_GFP"]="${pc}-p-128/Triples-p-P${pn}"
  ["MULTIPLICATION_TRIPLE_GF2N"]="${pc}-2-40/Triples-2-P${pn}"
)

# Provide parameters in MP-SPDZ "Player-Data" folder.
# Note that we always provide parameters for both prime fields and fields of
# characteristic 2 regardless of the tuple type requested for reasons of simplicity.
declare fields=("p" "2")
for f in "${fields[@]}"
do

  [[ "$f" = "p" ]] && bit_width="128" || bit_width="40"
	folder="Player-Data/${KII_PLAYER_COUNT}-${f}-${bit_width}"
	mkdir -p "${folder}"
  echo "Providing parameters for field ${f}-${bit_width} in folder ${folder}"

  # Write MAC key share
  macKeyShareFile="${folder}/Player-MAC-Keys-${f}-P${KII_PLAYER_NUMBER}"
  macKeyShare=$(cat "/etc/kii/secret-params/mac_key_share_${f}")
  echo "${KII_PLAYER_COUNT} ${macKeyShare}" > "${macKeyShareFile}"
  echo "MAC key share for player ${KII_PLAYER_NUMBER} written to ${macKeyShareFile}"

done

# Write player file containing CRG service endpoints
playerFile="players"
for (( i=0; i<pc; i++ ))
do
  endpointEnvName="KII_PLAYER_ENDPOINT_${i}"
  echo ${!endpointEnvName}
done >> ${playerFile}

# TODO Remove this as soon as we have something in place to ensure that the "network is ready"
sleep 10s

# Execute cowgear offline phase
cmd="cowgear-offline.x --player ${KII_PLAYER_NUMBER} --number-of-parties ${KII_PLAYER_COUNT} --playerfile ${playerFile} --tuple-count ${KII_TUPLES_PER_JOB} ${argsByType[${KII_TUPLE_TYPE}]} ${KII_PLAYER_COUNT}"
retry 5 eval "$cmd"

# Copy generated tuples to path expected by KII
cp "Player-Data/${tupleFileByType[${KII_TUPLE_TYPE}]}" "${KII_TUPLE_FILE}"
