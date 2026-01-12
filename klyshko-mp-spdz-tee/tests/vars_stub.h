/*
 * Stub version of vars.h for unit testing
 * Removes TEE/mbedtls dependencies to allow testing CRG.c without SGX hardware
 *
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#ifndef VARS_STUB_H
#define VARS_STUB_H

#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#ifndef EXTERN
#define EXTERN extern
#endif

// Variables that CRG.c uses (from vars.h)
EXTERN int other_player_number;
EXTERN int player_number_defined;
EXTERN int number_of_players;
EXTERN char* kii_job_id_defined;
EXTERN char** kii_endpoints;
EXTERN int base_port;

// Function declarations that CRG.c needs
EXTERN void box_out(const char *str);
EXTERN int ssl_client_setup_and_handshake();
EXTERN int ssl_server_setup_and_handshake();
EXTERN int local_attestation(char **Player_MAC_Keys_p, char **Player_MAC_Keys_2);

// Constants
#define KEY_LENGTH 128
#define DEBUG_LEVEL 1

#endif /* VARS_STUB_H */

