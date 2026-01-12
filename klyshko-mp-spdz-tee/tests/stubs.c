/*
 * Stub implementations for TEE dependencies
 * This allows testing CRG.c without requiring SGX hardware
 *
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

#include <stdio.h>
#include <stdlib.h>
#include "vars_stub.h"

// Note: Variables are defined in CRG.c (since it has #define EXTERN)
// We only need to provide stub function implementations here

// Stub function implementations
void box_out(const char *str) {
    (void)str;  // Suppress unused parameter warning
    // Do nothing - just satisfy the linker
}

int ssl_client_setup_and_handshake() {
    return 0;  // Return success
}

int ssl_server_setup_and_handshake() {
    return 0;  // Return success
}

int local_attestation(char **Player_MAC_Keys_p, char **Player_MAC_Keys_2) {
    (void)Player_MAC_Keys_p;   // Suppress unused parameter warning
    (void)Player_MAC_Keys_2;  // Suppress unused parameter warning
    return 0;  // Return success
}

