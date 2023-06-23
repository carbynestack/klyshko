#!/bin/bash

set -e

BASE_PORT=10000
PARTIES=2
FIELD_TYPE="gfp"     # One of gfp, gf2n
TUPLE_TYPE="squares" # One of bits, inverses, squares, triples
TUPLE_COUNT=$([[ $FIELD_TYPE = gfp ]] && echo 100000 || echo 1000)
FOLDER="pc_${PARTIES}-ft_${FIELD_TYPE}-tt_${TUPLE_TYPE}-tc_${TUPLE_COUNT}-bp_${BASE_PORT}"
EXECUTABLE=$(pwd)/../MP-SPDZ/cowgear-offline.x
PLAYERFILE="players"

echo "Running ${EXECUTABLE} in ${FOLDER}"

rm -rf "${FOLDER}"
mkdir -p "${FOLDER}"
pushd "${FOLDER}"

# Generate the playerfile
for ((i = 0; i < PARTIES; i++)); do
	echo "127.0.0.1:$((BASE_PORT + i))"
done >> ${PLAYERFILE}

for ((i = 0; i < PARTIES; i++)); do
	${EXECUTABLE} --number-of-parties ${PARTIES} --port $((BASE_PORT + i)) --player "${i}" --playerfile ${PLAYERFILE} \
        --tuple-count "${TUPLE_COUNT}" --field-type ${FIELD_TYPE} --tuple-type ${TUPLE_TYPE} &
	pids[${i}]=$!
done

# wait for all pids
for pid in ${pids[*]}; do
	wait $pid
done

popd
