#!/usr/bin/env bash
#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

#
# Deploys a 2-party low-gear offline phase setup.
#
set -e

BASE_PORT=10000
PARTIES=2
FIELD_TYPE="gfp"    # One of gfp, gf2n
TUPLE_TYPE="squares" # One of bits, inverses, squares, triples
TUPLE_COUNT=$([[ $FIELD_TYPE = gfp ]] && echo 500000 || echo 15000)
FOLDER="pc_${PARTIES}-ft_${FIELD_TYPE}-tt_${TUPLE_TYPE}-tc_${TUPLE_COUNT}-bp_${BASE_PORT}"
EXECUTABLE=$(pwd)/../../../MP-SPDZ/cowgear-offline.x
PLAYER_FILE="players"

echo "Running ${EXECUTABLE} in ${FOLDER}"

rm -rf "${FOLDER}"
mkdir -p "${FOLDER}"
pushd "${FOLDER}"

# Generate the playerfile
for ((i = 0; i < PARTIES; i++)); do
	echo "127.0.0.1:$((BASE_PORT + i))"
done >> ${PLAYER_FILE}

START=$(date +%s.%N)
for ((i = 0; i < PARTIES; i++)); do
  mkdir -p "${i}"
  pushd "${i}"
	${EXECUTABLE} --number-of-parties ${PARTIES} --port $((BASE_PORT + i)) --player "${i}" --playerfile "../${PLAYER_FILE}" \
        --tuple-count "${TUPLE_COUNT}" --field-type ${FIELD_TYPE} --tuple-type ${TUPLE_TYPE} &
	pids[${i}]=$!
	popd
done

# wait for all pids
for pid in ${pids[*]}; do
	wait $pid
done
END=$(date +%s.%N)

ELAPSED=$( echo "${END} - ${START}" | bc -l )
echo "Elapsed time: ${ELAPSED} seconds"

popd
