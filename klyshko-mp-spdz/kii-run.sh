#!/usr/bin/env bash
#
# Copyright (c) 2022 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

# Setup offline executable command line arguments dictionary
n=${KII_TUPLES_PER_JOB}
pn=${KII_PLAYER_NUMBER}
declare -A argsByType=(
  ["BIT_GFP"]="--nbits 0,${n}"
  ["BIT_GF2N"]="--nbits ${n},0"
  ["INPUT_MASK_GFP"]="--ntriples 0,${n}"
  ["INPUT_MASK_GF2N"]="--ntriples ${n},0"
  ["INVERSE_TUPLE_GFP"]="--ninverses ${n}"
  ["INVERSE_TUPLE_GF2N"]="--ninverses ${n}"
  ["SQUARE_TUPLE_GFP"]="--nsquares 0,${n}"
  ["SQUARE_TUPLE_GF2N"]="--nsquares ${n},0"
  ["MULTIPLICATION_TRIPLE_GFP"]="--ntriples 0,${n}"
  ["MULTIPLICATION_TRIPLE_GF2N"]="--ntriples ${n},0"
)
declare -A folderByType=(
  ["BIT_GFP"]="2-p-128/Bits-p-P${pn}"
  ["BIT_GF2N"]="2-2-40/Bits-2-P${pn}"
  ["INPUT_MASK_GFP"]="2-p-128/Triples-p-P${pn}"
  ["INPUT_MASK_GF2N"]="2-2-40/Triples-2-P${pn}"
  ["INVERSE_TUPLE_GFP"]="2-p-128/Inverses-p-P${pn}"
  ["INVERSE_TUPLE_GF2N"]="2-2-40/Inverses-2-P${pn}"
  ["SQUARE_TUPLE_GFP"]="2-p-128/Squares-p-P${pn}"
  ["SQUARE_TUPLE_GF2N"]="2-2-40/Squares-2-P${pn}"
  ["MULTIPLICATION_TRIPLE_GFP"]="2-p-128/Triples-p-P${pn}"
  ["MULTIPLICATION_TRIPLE_GF2N"]="2-2-40/Triples-2-P${pn}"
)

# Provide required parameters in MP-SPDZ "Player-Data" folder
prime=$(cat /etc/kii/params/prime)
declare fields=("p" "2")
for f in "${fields[@]}"
do

  [[ "$f" = "p" ]] && bit_width="128" || bit_width="40"
	folder="Player-Data/${KII_PLAYER_COUNT}-${f}-${bit_width}"
	mkdir -p "${folder}"
  echo "Providing parameters for field ${f}-${bit_width} in folder ${folder}"

  # Write MAC key shares
  for pn in $(seq 0 $((KII_PLAYER_COUNT-1)))
  do
    macKeyShareFile="${folder}/Player-MAC-Keys-${f}-P${pn}"
    if [[ ${pn} -eq ${KII_PLAYER_NUMBER} ]]; then
      src="/etc/kii/secret-params"
    else
      src="/etc/kii/extra-params"
    fi
    macKeyShare=$(cat "${src}/mac_key_share_${f}")
    echo "${KII_PLAYER_COUNT} ${macKeyShare}" > "${macKeyShareFile}"
    echo "MAC key share for player ${pn} written to ${macKeyShareFile}"
  done

done

# Execute offline phase
seed=$(echo "${KII_JOB_ID}" | md5sum)
cmd="./Fake-Offline.x -d 0 --prime ${prime} --prngseed ${seed:0:16} ${argsByType[${KII_TUPLE_TYPE}]} ${KII_PLAYER_COUNT}"
eval "$cmd"

# Copy generated tuples to path expected by KII
cp "Player-Data/${folderByType[${KII_TUPLE_TYPE}]}" "${KII_TUPLE_FILE}"
