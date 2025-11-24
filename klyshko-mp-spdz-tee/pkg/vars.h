/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#define _GNU_SOURCE
#include <assert.h>
#include <ctype.h>
#include <dlfcn.h>
#include <errno.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>  // For sleep() function

#include "mbedtls/build_info.h"
#include "mbedtls/ctr_drbg.h"
#include "mbedtls/debug.h"
#include "mbedtls/entropy.h"
#include "mbedtls/error.h"
#include "mbedtls/net_sockets.h"
#include "mbedtls/ssl.h"
#include "mbedtls/x509.h"
#include "ra_tls.h"
#include "secretsharing.pb-c.h"

#define DEBUG_LEVEL     1
#define MALICIOUS_STR   "MALICIOUS DATA"
#define mbedtls_fprintf fprintf
#define mbedtls_printf  printf

#define SRV_CRT_PATH "ssl/server.crt"
#define SRV_KEY_PATH "ssl/server.key"
// #define BASE_PORT    "4433"
#define SERVER_NAME          "localhost"
#define GET_REQUEST          "GET / HTTP/1.0\r\n\r\n"
#define MBEDTLS_EXIT_SUCCESS EXIT_SUCCESS
#define MBEDTLS_EXIT_FAILURE EXIT_FAILURE
#define MAX_MSG_SIZE         1024
#define RETRY_DELAY          1

#ifndef EXTERN
#define EXTERN extern
#endif

EXTERN int other_player_number;
EXTERN int player_number_defined;
EXTERN int number_of_players;
EXTERN char* kii_job_id_defined;

#define HTTP_RESPONSE                                    \
    "HTTP/1.0 200 OK\r\nContent-Type: text/html\r\n\r\n" \
    "<h2>mbed TLS Test Server</h2>\r\n"                  \
    "<p>Successful connection using: %s</p>\r\n"
    
EXTERN void box_out(const char *str);
EXTERN int ssl_client_setup_and_handshake();
EXTERN int ssl_server_setup_and_handshake();
// EXTERN void local_attestation();

EXTERN char** kii_endpoints;
EXTERN int base_port;

#define KEY_LENGTH 128
