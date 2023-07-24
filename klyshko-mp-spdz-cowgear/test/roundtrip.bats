#!/usr/bin/env bash
#
# Copyright (c) 2023 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

bats_require_minimum_version "1.10.0"

#######################################
# Generates the file tree for running a single player in the generation phase.
# Globals:
#   None
# Arguments:
#   The directory in which the file tree should be rooted in.
#   The overall number of players participating in the generation phase.
#   The zero-based number of the player for which to generate the file tree.
#   The prime number to be used.
#   The MAC key share for the prime field.
#   The MAC key share for the field of characteristic 2.
# Outputs:
#   None
#######################################
function create_generation_volume() {
    local dir=$1
    local player_count=$2
    local player=$3
    local prime=$4
    local mac_key_share_p=$5
    local mac_key_share_2=$6
    mkdir -p {"${dir}/${player}/params","${dir}/${player}/secret-params"}
    echo "${prime}" > "${dir}/${player}/params/prime"
    echo "${mac_key_share_p}" > "${dir}/${player}/secret-params/mac_key_share_p"
    echo "${mac_key_share_2}" > "${dir}/${player}/secret-params/mac_key_share_2"
}

#######################################
# Generates the file tree for running the generation phase.
# Globals:
#   None
# Arguments:
#   The directory in which the file tree should be rooted in.
#   The overall number of players participating in the generation phase.
#   The prime number to be used.
#   The MAC key shares for the prime field as an array containing one share per
#   party.
#   The MAC key shares for the field of characteristic 2 as an array containing
#   one share per party.
# Outputs:
#   None
#######################################
function create_generation_volumes() {
    declare -a argv=("${@}")
    local dir=${argv[0]}
    local player_count=${argv[1]}
    local prime=${argv[2]}
    local mac_key_shares_p=("${argv[@]:3:${player_count}}")
    local mac_key_shares_2=("${argv[@]:(3 + ${player_count}):${player_count}}")
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        create_generation_volume "${dir}" "${player_count}" "${pid}" "${prime}" "${mac_key_shares_p[pid]}" "${mac_key_shares_2[pid]}"
    done
}

#######################################
# Generates the Docker Compose service entry for a single player in the
# generation phase.
# Globals:
#   None
# Arguments:
#   The Docker Compose file to inject the service entry into.
#   The directory containing the configuration file tree to be attached to the
#   container created by Docker Compose for the player.
#   The zero-based number of the player for which the entry will be generated.
#   The overall number of players participating in the generation phase.
#   The number of tuples to be generated.
# Outputs:
#   None
#######################################
function create_generation_docker_compose_entry() {
    local compose_file=$1
    local volumes_dir=$2
    local player=$3
    local player_count=$4
    local tuple_count=$5
    local uid
    uid=$(id -u)
    local gid
    gid=$(id -g)
    yq "
        with(.services.player-${player};
            .image=\"mp-spdz-cowgear:latest\" |
            .user=\"${uid}:${gid}\" |
            .ports=[\"$((5000 + player)):5000\"] |
            .environment=[
                \"KII_JOB_ID=58301070-9b4c-4d38-ae89-d7aa989720a0\",
                \"KII_TUPLES_PER_JOB=${tuple_count}\",
                \"KII_PLAYER_NUMBER=${player}\",
                \"KII_PLAYER_COUNT=${player_count}\",
                \"KII_TUPLE_TYPE=MULTIPLICATION_TRIPLE_GFP\",
                \"KII_TUPLE_FILE=/etc/kii/tuples\"
            ] |
            .volumes=[
                \"./${volumes_dir}/${player}:/etc/kii/\",
                \"/etc/passwd:/etc/passwd:ro\",
                \"/etc/group:/etc/group:ro\"
            ]
        )
    " -i "${compose_file}"
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        yq ".services.player-${player}.environment += \"KII_PLAYER_ENDPOINT_${pid}=player-${pid}:$((5000 + pid))\"" -i "${compose_file}"
    done
}

#######################################
# Generates the Docker Compose file for running the generation phase.
# Globals:
#   None
# Arguments:
#   The directory containing the configuration file tree to be attached to the
#   containers generated for the players.
#   The overall number of players participating in the generation phase.
#   The number of tuples to be generated.
# Outputs:
#   Writes the file path of the generated Docker Compose file to STDOUT.
#######################################
function create_generation_docker_compose() {
    local volumes_dir=$1
    local player_count=$2
    local tuple_count=$3
    local compose_file="docker-compose-generation.yaml"
    rm -f ${compose_file}
    touch ${compose_file}
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        create_generation_docker_compose_entry "${compose_file}" "${volumes_dir}" "${pid}" "${player_count}" "${tuple_count}"
    done
    echo ${compose_file}
}

#######################################
# Generates the MPC program used for validating the generated tuples and a
# script to execute it.
# Globals:
#   None
# Arguments:
#   The directory to which the MPC program should be written to.
#   The overall number of players participating in the validation phase.
# Outputs:
#   None
#######################################
function create_validation_script() {
    dir=$1
    player_count=$2
    mkdir -p "${dir}/shared"
    cat <<- EOF > "${dir}/shared/run.sh"
#!/usr/bin/env bash

cat << EOF_MPC > test.mpc
a = sint(1)
b = sint(2)
c = a * b
print_ln('Result: %s', c.reveal())
EOF_MPC

./compile.py test.mpc
./Player-Online.x -p "\${PLAYER_ID}" -N "${player_count}" test -h "player-0"
EOF
    chmod +x "${dir}/shared/run.sh"
}

#######################################
# Generates the file tree for running a single player in the validation phase.
# Globals:
#   None
# Arguments:
#   The directory in which the file tree should be rooted in.
#   The overall number of players participating in the validation phase.
#   The zero-based number of the player for which to generate the file tree.
#   The prime number to be used.
#   The MAC key share for the prime field.
#   The path of the triple file to be copied over the validation environment.
# Outputs:
#   None
#######################################
function create_validation_volume() {
    local dir=$1
    local player_count=$2
    local player=$3
    local prime=$4
    local mac_key_share_p=$5
    local triple_file=$6
    mkdir -p "${dir}/${player}/2-p-128"
    echo "${prime}" > "${dir}/${player}/2-p-128/Params-Data"
    cat <<- EOF > "${dir}/${player}/2-p-128/Player-MAC-Keys-p-P${player}"
${player_count}
${mac_key_share_p}
EOF
    cp "${triple_file}" "${dir}/${player}/2-p-128/Triples-p-P${player}"
}

#######################################
# Generates the file tree for running the validation phase.
# Globals:
#   None
# Arguments:
#   The directory in which the file tree should be rooted in.
#   The overall number of players participating in the validation phase.
#   The prime number to be used.
#   The MAC key shares for the prime field.
#   The triple files to be copied over to the validation environment.
# Outputs:
#   None
#######################################
function create_validation_volumes() {
    declare -a argv=("${@}")
    local dir=${argv[0]}
    local player_count=${argv[1]}
    local prime=${argv[2]}
    local mac_key_shares_p=("${argv[@]:3:${player_count}}")
    local triple_files=("${argv[@]:(3 + ${player_count}):${player_count}}")
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        create_validation_volume "${dir}" "${player_count}" "${pid}" "${prime}" "${mac_key_shares_p[pid]}" "${triple_files[pid]}"
    done
    create_validation_script "${dir}" "${player_count}"
}

#######################################
# Generates the Docker Compose service entry for a single player in the validation phase.
# Globals:
#   None
# Arguments:
#   The Docker Compose file to inject the service entry into.
#   The directory containing the configuration file tree to be attached to the container generated for the player.
#   The zero-based number of the player for which the entry will be generated.
# Outputs:
#   None
#######################################
function create_validation_docker_compose_entry() {
    local compose_file=$1
    local volumes_dir=$2
    local player=$3
    local port=$((5000 + player))
    local uid
    uid=$(id -u)
    local gid
    gid=$(id -g)
    yq "
        with(.services.player-${player};
            .image=\"ghcr.io/carbynestack/spdz:642d11f_no-offline\" |
            .user=\"${uid}:${gid}\" |
            .ports=[\"${port}:${port}\"] |
            .environment=[
                \"PLAYER_ID=${player}\"
            ] |
            .entrypoint=[
                \"/shared/run.sh\"
            ] |
            .volumes=[
                \"./${volumes_dir}/shared:/shared/\",
                \"./${volumes_dir}/${player}:/mp-spdz-642d11f_no-offline/Player-Data\",
                \"/etc/passwd:/etc/passwd:ro\",
                \"/etc/group:/etc/group:ro\"
            ]
        )
    " -i "${compose_file}"
}

#######################################
# Generates the Docker Compose file for running the validation phase.
# Globals:
#   None
# Arguments:
#   The directory containing the configuration file tree to be attached to the containers generated for the players.
#   The overall number of players participating in the validation phase.
# Outputs:
#   Writes the file path of the generated Docker Compose file to STDOUT.
#######################################
function create_validation_docker_compose {
    local volumes_dir=$1
    local player_count=$2
    local compose_file="docker-compose-validation.yaml"
    rm -f ${compose_file}
    touch ${compose_file}
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        create_validation_docker_compose_entry "${compose_file}" "${volumes_dir}" "${pid}"
    done
    echo ${compose_file}
}

#######################################
# Initializes BATS by loading the required support libaries.
# Globals:
#   None
# Arguments:
#   None
# Outputs:
#   None
#######################################
function init_bats() {
    load 'test_helper/bats-support/load'
    load 'test_helper/bats-assert/load'
}

#######################################
# Setup logic that is run before each test.
# Globals:
#   None
# Arguments:
#   None
# Outputs:
#   None
#######################################
function setup() {
    init_bats
}

#######################################
# First runs a generation phase in which triples are generated using the
# Cow-Gear CRG and then validates the generated triples by running the MP-SPDZ
# online phase in which the triples are consumed.
# Globals:
#   None
# Arguments:
#   None
# Outputs:
#   None
#######################################
@test "2-party End-to-End Tuple Generation/Consumption Test" {

    # Define parameters
    local player_count=2
    local tuple_count=500000
    local prime=198766463529478683931867765928436695041
    declare -a mac_key_shares_p
    mac_key_shares_p[0]=-88222337191559387830816715872691188861
    mac_key_shares_p[1]=1113507028231509545156335486838233835
    declare -a mac_key_shares_2
    mac_key_shares_2[0]=0xb660b323e6
    mac_key_shares_2[1]=0x4ec9a0343c

    # Phase I: Generate the tuples using the Klyshko Cow-Gear CRG

    # Generate the tuples
    local generation_dir="generation"
    create_generation_volumes "${generation_dir}" "${player_count}" "${prime}" "${mac_key_shares_p[@]}" "${mac_key_shares_2[@]}"
    local generation_compose_file
    generation_compose_file=$(create_generation_docker_compose ${generation_dir} ${player_count} ${tuple_count})
    run docker-compose -f "${generation_compose_file}" up

    # Check that docker compose did not fail
    assert_success

    # Validate that no player failed
    local pid
    for (( pid=0; pid<player_count; pid++ )); do
        assert_output --partial "test_player-${pid}_1 exited with code 0"
    done

    # Validate that tuple files have been created and that the size is as expected
    for (( pid=0; pid<player_count; pid++ )); do
        assert [ -e "${generation_dir}/${pid}/tuples" ]
        local tuple_file_size
        tuple_file_size=$(stat -c %s ${generation_dir}/${pid}/tuples)
        local expected_size
        expected_size=$(( 37 + 3 * 2 * 16 * tuple_count )) # Header: 37, tuple elements: 3, value+mac: 2, 128-bit: 16
        assert_equal "${tuple_file_size}" "${expected_size}"
    done

    # Phase II: Validate the tuples generated in phase I by using them in the online phase

    declare -a triple_files
    for (( pid=0; pid<player_count; pid++ )); do
        triple_files[pid]="${generation_dir}/${pid}/tuples"
    done
    local validation_dir="validation"
    create_validation_volumes "${validation_dir}" "${player_count}" "${prime}" "${mac_key_shares_p[@]}" "${triple_files[@]}"
    local validation_compose_file
    validation_compose_file=$(create_validation_docker_compose ${validation_dir} ${player_count})
    run docker-compose -f "${validation_compose_file}" up

    # Check that docker compose did not fail
    assert_success

    # Validate that the correct result has been computer
    assert_output --partial "Result: 2"

    # Validate that no MAC check error occured
    refute_output --partial "MacCheck Failure"

    # Clean-up
    rm -rf "${generation_dir}" "${generation_compose_file}" "${validation_dir}" "${validation_compose_file}"

}
