#!/usr/bin/env bash
#
# Copyright (c) 2025 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e
if [ -f "kii_0.log" ]; then
    rm "kii_0.log"
fi

if [ -f "kii_1.log" ]; then
    rm "kii_1.log"
fi

# Configuration Variables
export BASE_PORT="4433"

# Retrieve mr_enclave and mr_signer values from server.sig
output=$(gramine-sgx-sigstruct-view server.sig)
mr_enclave=$(echo "$output" | grep "mr_enclave" | awk '{print $2}')
mr_signer=$(echo "$output" | grep "mr_signer" | awk '{print $2}')


# Check if mr_enclave and mr_signer are correctly retrieved
if [ -z "$mr_enclave" ] || [ -z "$mr_signer" ]; then
    echo "Error: Could not retrieve mr_enclave or mr_signer from server.sig"
    exit 1
fi

# Set required RA-TLS verification variables
export RA_TLS_MRSIGNER="$mr_signer"
export RA_TLS_MRENCLAVE="$mr_enclave"
export RA_TLS_ISV_SVN="any"
export RA_TLS_ISV_PROD_ID="any"

if [ "$KII_PLAYER_NUMBER" -eq 0 ]; then
    export KII_PLAYER_NAME="APOLLO"
elif [ "$KII_PLAYER_NUMBER" -eq 1 ]; then
    export KII_PLAYER_NAME="STARBUCK"
else
    echo "Error: Invalid KII_PLAYER_NUMBER. Must be 0 or 1."
    exit 1
fi

export RA_TLS_ALLOW_DEBUG_ENCLAVE_INSECURE=1
export RA_TLS_ALLOW_OUTDATED_TCB_INSECURE=1
export RA_TLS_ALLOW_HW_CONFIG_NEEDED=1
export RA_TLS_ALLOW_SW_HARDENING_NEEDED=1

box_out() {
    local text="PLAYER $KII_PLAYER_NUMBER $KII_PLAYER_NAME $1" # Text to display inside the box
    local box_color="\e[43m" # Yellow background
    local text_color="\e[32m" # Green text
    local reset_color="\e[0m" # Reset to default

    local padding=2
    printf "%s " "${box_color}"
    printf "%.0s " $(seq 1 $padding)
    printf "${text_color}%s${reset_color}" "$text"
    printf "%.0s " $(seq 1 $padding)
    printf "%s " "${reset_color}"
}


box_out "[0] Starting execution for player $KII_PLAYER_NUMBER $KII_PLAYER_NAME"


echo "Starting player $KII_PLAYER_NUMBER $KII_PLAYER_NAME with enclave mr_enclave: $mr_enclave and mr_signer: $mr_signer" > "player_${KII_PLAYER_NUMBER}.log"

box_out "[1] Spawning TEE.."

gramine-sgx ./server "$mr_enclave" "$mr_signer" 0 0    &
server_pid=$!
./KII "$mr_enclave" "$mr_signer" 0 0 "$KII_PLAYER_NUMBER"  &


wait $server_pid
box_out "[8] Copied Correlated Randomness to /kii/tuples.."
