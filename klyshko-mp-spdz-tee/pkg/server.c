/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#include "vars.h"

void box_out(const char *str)
{
    // ANSI escape code for black text on green background
    mbedtls_printf("\033[30;42m"); // 30: black text, 42: green background
    mbedtls_printf("%s", str);     // Print the string
    mbedtls_printf("\033[0m\n");   // Reset to default colors
}

int (*ra_tls_verify_callback_extended_der_f)(uint8_t *der_crt, size_t der_crt_size,
                                             struct ra_tls_verify_callback_results *results);

/* RA-TLS: if specified in command-line options, use our own callback to verify SGX measurements */
void (*ra_tls_set_measurement_callback_f)(int (*f_cb)(const char *mrenclave, const char *mrsigner,
                                                      const char *isv_prod_id,
                                                      const char *isv_svn));

static int parse_hex(const char *hex, void *buffer, size_t buffer_size)
{
    // Use strnlen to safely check string length and prevent over-read if not null-terminated
    // Use a reasonable maximum (expected length + some margin) to detect non-null-terminated strings
    size_t max_len = buffer_size * 2 + 10;
    size_t hex_len = strnlen(hex, max_len);
    
    // Check if string length matches expected length
    // If hex_len equals max_len, the string is longer than expected or not null-terminated
    if (hex_len != buffer_size * 2 || hex_len == max_len)
        return -1;

    for (size_t i = 0; i < buffer_size; i++)
    {
        if (!isxdigit(hex[i * 2]) || !isxdigit(hex[i * 2 + 1]))
            return -1;
        sscanf(hex + i * 2, "%02hhx", &((uint8_t *)buffer)[i]);
    }
    return 0;
}

int verify_player_details(char *kii_job_id, int received_other_player_number,
                          char *kii_job_id_defined, int other_player_number)
{
    // printf("received kii_job_id: %s\n", kii_job_id);
    // printf("kii_job_id_defined: %s\n", kii_job_id_defined);
    // printf("strcmp result of job id: %d\n", strcmp(kii_job_id, kii_job_id_defined));
    // printf("received_other_player_number: %d\n", received_other_player_number);
    // printf("other_player_number: %d\n", other_player_number);
    if (strcmp(kii_job_id, kii_job_id_defined) == 0 &&
        received_other_player_number == other_player_number)
    {
        return 0;
    }
    else
    {
        return -1;
    }
}

/* expected SGX measurements in binary form */
static char g_expected_mrenclave[32];
static char g_expected_mrsigner[32];
static char g_expected_isv_prod_id[2];
static char g_expected_isv_svn[2];

static bool g_verify_mrenclave = false;
static bool g_verify_mrsigner = false;
static bool g_verify_isv_prod_id = false;
static bool g_verify_isv_svn = false;

/* RA-TLS: our own callback to verify SGX measurements */
static int my_verify_measurements(const char *mrenclave, const char *mrsigner,
                                  const char *isv_prod_id, const char *isv_svn)
{
    assert(mrenclave && mrsigner && isv_prod_id && isv_svn);

    if (g_verify_mrenclave && memcmp(mrenclave, g_expected_mrenclave, sizeof(g_expected_mrenclave)))
        return -1;

    if (g_verify_mrsigner && memcmp(mrsigner, g_expected_mrsigner, sizeof(g_expected_mrsigner)))
        return -1;

    if (g_verify_isv_prod_id &&
        memcmp(isv_prod_id, g_expected_isv_prod_id, sizeof(g_expected_isv_prod_id)))
        return -1;

    if (g_verify_isv_svn && memcmp(isv_svn, g_expected_isv_svn, sizeof(g_expected_isv_svn)))
        return -1;

    return 0;
}

/* RA-TLS: mbedTLS-specific callback to verify the x509 certificate */
static int my_verify_callback(void *data, mbedtls_x509_crt *crt, int depth, uint32_t *flags)
{
    if (depth != 0)
    {
        /* the cert chain in RA-TLS consists of single self-signed cert, so we expect depth 0 */
        return MBEDTLS_ERR_X509_INVALID_FORMAT;
    }
    if (flags)
    {
        /* mbedTLS sets flags to signal that the cert is not to be trusted (e.g., it is not
         * correctly signed by a trusted CA; since RA-TLS uses self-signed certs, we don't care
         * what mbedTLS thinks and ignore internal cert verification logic of mbedTLS */
        *flags = 0;
    }
    return ra_tls_verify_callback_extended_der_f(crt->raw.p, crt->raw.len,
                                                 (struct ra_tls_verify_callback_results *)data);
}


static void my_debug(void *ctx, int level, const char *file, int line, const char *str)
{
    ((void)level);

    mbedtls_fprintf((FILE *)ctx, "%s:%04d: %s\n", file, line, str);
    fflush((FILE *)ctx);
}

static ssize_t file_read(const char *path, char *buf, size_t count)
{
    FILE *f = fopen(path, "r");
    if (!f)
        return -errno;

    ssize_t bytes = fread(buf, 1, count, f);
    if (bytes <= 0)
    {
        int errsv = errno;
        fclose(f);
        return -errsv;
    }

    int close_ret = fclose(f);
    if (close_ret < 0)
        return -errno;

    return bytes;
}

char *addHex2(const char *hex1, const char *hex2)
{
    // Convert hex strings to unsigned long long integers
    unsigned long long num1 = strtoull(hex1, NULL, 16);
    unsigned long long num2 = strtoull(hex2, NULL, 16);

    // Add the two numbers
    unsigned long long sum = num1 + num2;

    // Allocate memory for the result string (16 characters + 1 for null terminator)
    char *result = (char *)malloc(17);
    if (result == NULL)
    {
        return NULL; // Handle memory allocation failure
    }

    // Convert the sum back to a hexadecimal string
    snprintf(result, 17, "%016llx", sum);

    return result;
}

int ssl_server_setup_and_handshake(char *a, char *b, char *c, char *d, char *Player_MAC_Keys_p[],
                                   char *Player_MAC_Keys_2[], char *Seed)
{

    // box_out("C entry point.\n");
    printf("Player number %d acting as server \n", player_number_defined);
    int no_of_parameters = 5;
    int ret;
    size_t len;
    mbedtls_net_context listen_fd;
    mbedtls_net_context client_fd;
    unsigned char buf[1024];
    const char *pers = "ssl_server";
    char *error;

    //***
    char server_port[10];
    char server_ip[20];

    //****$$$$****//
    int exit_code = MBEDTLS_EXIT_FAILURE;
    uint32_t flags;
    void *ra_tls_verify_lib = NULL;
    ra_tls_verify_callback_extended_der_f = NULL;
    ra_tls_set_measurement_callback_f = NULL;
    struct ra_tls_verify_callback_results my_verify_callback_results = {0};
    //****$$$$****//

    void *ra_tls_attest_lib;
    int (*ra_tls_create_key_and_crt_der_f)(uint8_t **der_key, size_t *der_key_size,
                                           uint8_t **der_crt, size_t *der_crt_size);

    uint8_t *der_key = NULL;
    uint8_t *der_crt = NULL;

    mbedtls_entropy_context entropy;
    mbedtls_ctr_drbg_context ctr_drbg;
    mbedtls_ssl_context ssl;
    mbedtls_ssl_config conf;
    mbedtls_x509_crt srvcert;
    mbedtls_pk_context pkey;

    mbedtls_net_init(&listen_fd);
    mbedtls_net_init(&client_fd);
    mbedtls_ssl_init(&ssl);
    mbedtls_ssl_config_init(&conf);
    mbedtls_x509_crt_init(&srvcert);
    mbedtls_pk_init(&pkey);
    mbedtls_entropy_init(&entropy);
    mbedtls_ctr_drbg_init(&ctr_drbg);

#if defined(MBEDTLS_DEBUG_C)
    mbedtls_debug_set_threshold(DEBUG_LEVEL);
#endif

    //****$$$$****//
    ra_tls_verify_lib = dlopen("libra_tls_verify_dcap_gramine.so", RTLD_LAZY);
    if (!ra_tls_verify_lib)
    {
        mbedtls_printf("%s\n", dlerror());
        mbedtls_printf(
            "User requested RA-TLS verification with DCAP inside SGX but cannot find lib\n");
        mbedtls_printf("Please make sure that you are using client_dcap.manifest\n");
        return 1;
    }

    if (ra_tls_verify_lib)
    {
        ra_tls_verify_callback_extended_der_f =
            dlsym(ra_tls_verify_lib, "ra_tls_verify_callback_extended_der");
        if ((error = dlerror()) != NULL)
        {
            mbedtls_printf("%s\n", error);
            return 1;
        }

        ra_tls_set_measurement_callback_f =
            dlsym(ra_tls_verify_lib, "ra_tls_set_measurement_callback");
        if ((error = dlerror()) != NULL)
        {
            mbedtls_printf("%s\n", error);
            return 1;
        }
    }

    if (no_of_parameters > 2 && ra_tls_verify_lib)
    {
        if (no_of_parameters != 5)
        {
            mbedtls_printf(
                "USAGE: %s %s <expected mrenclave> <expected mrsigner>"
                " <expected isv_prod_id> <expected isv_svn>\n"
                "       (first two in hex, last two as decimal; set to 0 to ignore)\n",
                a, b);
            return 1;
        }

        mbedtls_printf(
            "[ using our own SGX-measurement verification callback"
            " (via command line options) ]\n");

        g_verify_mrenclave = true;
        g_verify_mrsigner = true;
        g_verify_isv_prod_id = true;
        g_verify_isv_svn = true;

        (*ra_tls_set_measurement_callback_f)(my_verify_measurements);

        if (!strcmp(a, "0"))
        {
            mbedtls_printf("  - ignoring MRENCLAVE\n");
            g_verify_mrenclave = false;
        }
        else if (parse_hex(a, g_expected_mrenclave, sizeof(g_expected_mrenclave)) < 0)
        {
            mbedtls_printf("Cannot parse MRENCLAVE!\n");
            return 1;
        }

        if (!strcmp(b, "0"))
        {
            mbedtls_printf("  - ignoring MRSIGNER\n");
            g_verify_mrsigner = false;
        }
        else if (parse_hex(b, g_expected_mrsigner, sizeof(g_expected_mrsigner)) < 0)
        {
            mbedtls_printf("Cannot parse MRSIGNER!\n");
            return 1;
        }

        if (!strcmp(c, "0"))
        {
            mbedtls_printf("  - ignoring ISV_PROD_ID\n");
            g_verify_isv_prod_id = false;
        }
        else
        {
            errno = 0;
            uint16_t isv_prod_id = (uint16_t)strtoul(c, NULL, 10);
            if (errno)
            {
                mbedtls_printf("Cannot parse ISV_PROD_ID!\n");
                return 1;
            }
            // Validate buffer size before memcpy to prevent buffer overflow
            if (sizeof(g_expected_isv_prod_id) >= sizeof(isv_prod_id))
            {
                memcpy(g_expected_isv_prod_id, &isv_prod_id, sizeof(isv_prod_id));
            }
            else
            {
                mbedtls_printf("Error: Destination buffer too small for ISV_PROD_ID\n");
                return 1;
            }
        }

        if (!strcmp(d, "0"))
        {
            mbedtls_printf("  - ignoring ISV_SVN\n");
            g_verify_isv_svn = false;
        }
        else
        {
            errno = 0;
            uint16_t isv_svn = (uint16_t)strtoul(d, NULL, 10);
            if (errno)
            {
                mbedtls_printf("Cannot parse ISV_SVN\n");
                return 1;
            }
            // Validate buffer size before memcpy to prevent buffer overflow
            if (sizeof(g_expected_isv_svn) >= sizeof(isv_svn))
            {
                memcpy(g_expected_isv_svn, &isv_svn, sizeof(isv_svn));
            }
            else
            {
                mbedtls_printf("Error: Destination buffer too small for ISV_SVN\n");
                return 1;
            }
        }
    }

    mbedtls_ssl_conf_authmode(&conf, MBEDTLS_SSL_VERIFY_OPTIONAL);
    if (ra_tls_verify_lib)
    {
        /* use RA-TLS verification callback; this will overwrite CA chain set up above */
        mbedtls_printf("  . Installing RA-TLS callback ...");
        mbedtls_ssl_conf_verify(&conf, &my_verify_callback, &my_verify_callback_results);
        mbedtls_printf(" ok\n");
    }

    //****$$$$****//

    char attestation_type_str[32] = {0};
    ret = file_read("/dev/attestation/attestation_type", attestation_type_str,
                    sizeof(attestation_type_str) - 1);
    if (ret < 0 && ret != -ENOENT)
    {
        mbedtls_printf(
            "User requested RA-TLS attestation but cannot read SGX-specific file "
            "/dev/attestation/attestation_type\n");
        return 1;
    }

    if (ret == -ENOENT || !strcmp(attestation_type_str, "none"))
    {
        ra_tls_attest_lib = NULL;
        ra_tls_create_key_and_crt_der_f = NULL;
    }
    else if (!strcmp(attestation_type_str, "dcap"))
    {
        ra_tls_attest_lib = dlopen("libra_tls_attest.so", RTLD_LAZY);
        if (!ra_tls_attest_lib)
        {
            mbedtls_printf("User requested RA-TLS attestation but cannot find lib\n");
            return 1;
        }

        char *error;
        ra_tls_create_key_and_crt_der_f = dlsym(ra_tls_attest_lib, "ra_tls_create_key_and_crt_der");
        if ((error = dlerror()) != NULL)
        {
            mbedtls_printf("%s\n", error);
            return 1;
        }
    }
    else
    {
        mbedtls_printf("Unrecognized remote attestation type: %s\n", attestation_type_str);
        return 1;
    }

    mbedtls_printf("  . Seeding the random number generator...");
    fflush(stdout);

    // Use strnlen to safely get string length and prevent over-read if not null-terminated
    size_t pers_len = strnlen(pers, 64);
    ret = mbedtls_ctr_drbg_seed(&ctr_drbg, mbedtls_entropy_func, &entropy,
                                (const unsigned char *)pers, pers_len);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ctr_drbg_seed returned %d\n", ret);
        goto exit;
    }

    mbedtls_printf(" ok\n");

    if (ra_tls_attest_lib)
    {
        mbedtls_printf(
            "\n  . Creating the RA-TLS server cert and key (using \"%s\" as "
            "attestation type)...",
            attestation_type_str);
        fflush(stdout);

        size_t der_key_size;
        size_t der_crt_size;

        ret = (*ra_tls_create_key_and_crt_der_f)(&der_key, &der_key_size, &der_crt, &der_crt_size);
        if (ret != 0)
        {
            mbedtls_printf(" failed\n  !  ra_tls_create_key_and_crt_der returned %d\n\n", ret);
            goto exit;
        }

        ret = mbedtls_x509_crt_parse(&srvcert, (unsigned char *)der_crt, der_crt_size);
        if (ret != 0)
        {
            mbedtls_printf(" failed\n  !  mbedtls_x509_crt_parse returned %d\n\n", ret);
            goto exit;
        }

        ret = mbedtls_pk_parse_key(&pkey, (unsigned char *)der_key, der_key_size, /*pwd=*/NULL, 0,
                                   mbedtls_ctr_drbg_random, &ctr_drbg);
        if (ret != 0)
        {
            mbedtls_printf(" failed\n  !  mbedtls_pk_parse_key returned %d\n\n", ret);
            goto exit;
        }
    }

    mbedtls_printf(" ok\n");

    char *ip_address = kii_endpoints[player_number_defined];
    const char *colon_pos = strrchr(kii_endpoints[player_number_defined], ':');
    size_t ip_length = colon_pos - ip_address;
    if (colon_pos != NULL)
    {
        // Validate buffer sizes to prevent overflow
        if (ip_length >= sizeof(server_ip))
        {
            fprintf(stderr, "Error: IP address too long (max %zu characters)\n", sizeof(server_ip) - 1);
            goto exit;
        }
        
        // Use snprintf instead of strncpy for safe, null-terminated copying
        int port_len = snprintf(server_port, sizeof(server_port), "%.4s", colon_pos + 1);
        int ip_len = snprintf(server_ip, sizeof(server_ip), "%.*s", (int)ip_length, ip_address);
        
        if (port_len < 0 || ip_len < 0 || port_len >= (int)sizeof(server_port) || ip_len >= (int)sizeof(server_ip))
        {
            fprintf(stderr, "Error: Failed to copy IP address or port\n");
            goto exit;
        }
    }

    mbedtls_printf("  . Bind on https://%s:%s/ ...", server_ip, server_port);
    fflush(stdout);

    ret = mbedtls_net_bind(&listen_fd, NULL, server_port, MBEDTLS_NET_PROTO_TCP);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_net_bind returned %d\n\n", ret);
        goto exit;
    }

    mbedtls_printf(" ok\n");

    mbedtls_printf("  . Setting up the SSL data....");
    fflush(stdout);

    ret = mbedtls_ssl_config_defaults(&conf, MBEDTLS_SSL_IS_SERVER, MBEDTLS_SSL_TRANSPORT_STREAM,
                                      MBEDTLS_SSL_PRESET_DEFAULT);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_config_defaults returned %d\n\n", ret);
        goto exit;
    }

    mbedtls_ssl_conf_rng(&conf, mbedtls_ctr_drbg_random, &ctr_drbg);
    mbedtls_ssl_conf_dbg(&conf, my_debug, stdout);

    ret = mbedtls_ssl_conf_own_cert(&conf, &srvcert, &pkey);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_conf_own_cert returned %d\n\n", ret);
        goto exit;
    }

    ret = mbedtls_ssl_setup(&ssl, &conf);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_setup returned %d\n\n", ret);
        goto exit;
    }

    mbedtls_printf(" ok\n");
reset:
    if (other_player_number >= player_number_defined)
    {
        goto exit;
    }

    mbedtls_net_free(&client_fd);

    mbedtls_ssl_session_reset(&ssl);

    mbedtls_printf("  . Waiting for a remote connection ...");
    fflush(stdout);

    ret = mbedtls_net_accept(&listen_fd, &client_fd, NULL, 0, NULL);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_net_accept returned %d\n\n", ret);
        goto exit;
    }

    mbedtls_ssl_set_bio(&ssl, &client_fd, mbedtls_net_send, mbedtls_net_recv, NULL);

    mbedtls_printf(" ok\n");

    mbedtls_printf("  . Performing the SSL/TLS handshake...");
    fflush(stdout);

    while ((ret = mbedtls_ssl_handshake(&ssl)) != 0)
    {
        if (ret != MBEDTLS_ERR_SSL_WANT_READ && ret != MBEDTLS_ERR_SSL_WANT_WRITE)
        {
            mbedtls_printf(" failed\n  ! mbedtls_ssl_handshake returned -0x%x\n", -ret);
            mbedtls_printf(
                "  ! ra_tls_verify_callback_results:\n"
                "    attestation_scheme=%d, err_loc=%d, \n",
                my_verify_callback_results.attestation_scheme,
                my_verify_callback_results.err_loc);
            switch (my_verify_callback_results.attestation_scheme)
            {
            case RA_TLS_ATTESTATION_SCHEME_DCAP:
                mbedtls_printf(
                    "    dcap.func_verify_quote_result=0x%x, "
                    "dcap.quote_verification_result=0x%x\n\n",
                    my_verify_callback_results.dcap.func_verify_quote_result,
                    my_verify_callback_results.dcap.quote_verification_result);
                break;
            default:
                mbedtls_printf("  ! unknown attestation scheme!\n\n");
                break;
            }

            goto exit;
        }
    }

    mbedtls_printf(" ok\n");

    ret = mbedtls_ssl_get_verify_result(&ssl);
    if (ret != 0)
    {
        char vrfy_buf[512];
        mbedtls_printf(" failed\n");
        mbedtls_x509_crt_verify_info(vrfy_buf, sizeof(vrfy_buf), "  ! ", flags);
        mbedtls_printf("%s\n", vrfy_buf);

        /* verification failed for whatever reason, fail loudly */
        goto exit;
    }
    else
    {
        mbedtls_printf(" Step 4 Mutual Attestation between TEEs succeeded \n");
        box_out(" [4] Mutual Attestation between TEEs successful. \n");
    }

    fflush(stdout);
    PlayerInfo *msg;
    uint8_t secretbuff[MAX_MSG_SIZE];
    size_t play_len;

    do
    {
        play_len = sizeof(secretbuff) - 1;
        memset(buf, 0, sizeof(secretbuff));
        ret = mbedtls_ssl_read(&ssl, secretbuff, play_len);

        if (ret == MBEDTLS_ERR_SSL_WANT_READ || ret == MBEDTLS_ERR_SSL_WANT_WRITE)
            continue;

        if (ret <= 0)
        {
            switch (ret)
            {
            case MBEDTLS_ERR_SSL_PEER_CLOSE_NOTIFY:
                mbedtls_printf(" connection was closed gracefully\n");
                break;

            case MBEDTLS_ERR_NET_CONN_RESET:
                mbedtls_printf(" connection was reset by peer\n");
                break;

            default:
                mbedtls_printf(" mbedtls_ssl_read returned -0x%x\n", -ret);
                break;
            }

            break;
        }

        play_len = ret;
        mbedtls_printf(" %ld bytes read: \n", play_len);

        if (ret > 0)
            break;
    } while (1);

    msg = player_info__unpack(NULL, play_len, secretbuff);
    if (msg == NULL)
    {
        fprintf(stderr, "Error unpacking incoming message\n");
    }

    ret = verify_player_details(msg->kii_job_id, msg->player_number, kii_job_id_defined,
                                other_player_number);
    if (ret == -1)
    {
        printf("KII_JOB_ID or PLAYER_ID not valid.\n");
        int rcd = mbedtls_ssl_close_notify(&ssl);
        while (rcd < 0)
        {
            rcd = mbedtls_ssl_close_notify(&ssl);
        }
        return ret;
    }
    printf(" Step 5 Other player number : %d  player number and Job ID is verified and correct\n", other_player_number);
    box_out("[5] Job ID + Player number received and verified.\n");
    // code for sending the macshares and seed values from the server to client side
    SecretShare secret_message = SECRET_SHARE__INIT;
    secret_message.mackeyshare_2 = Player_MAC_Keys_2[player_number_defined];
    secret_message.mackeyshare_p = Player_MAC_Keys_p[player_number_defined];
    secret_message.seeds = Seed;
    unsigned lenth = secret_share__get_packed_size(&secret_message);
    if (lenth == 0)
    {
        fprintf(stderr, "packing or serialization error");
    }
    void *secret_buffer = malloc(lenth);
    if (!secret_buffer)
    {
        fprintf(stderr, "Memory allocation error\n");
    }

    secret_share__pack(&secret_message, secret_buffer);
    fprintf(stderr, "Writing %d serialized bytes\n", lenth);
    while ((ret = mbedtls_ssl_write(&ssl, secret_buffer, lenth)) <= 0)
    {
        if (ret == MBEDTLS_ERR_NET_CONN_RESET)
        {
            mbedtls_printf(" failed\n  ! peer closed the connection\n\n");
            goto exit;
        }

        if (ret != MBEDTLS_ERR_SSL_WANT_READ && ret != MBEDTLS_ERR_SSL_WANT_WRITE)
        {
            mbedtls_printf(" failed\n  ! mbedtls_ssl_write returned %d\n\n", ret);
            goto exit;
        }
    }

    lenth = ret;
    mbedtls_printf(" %d bytes written\n", lenth);

    fflush(stdout);

    // code for macshares and reading
    uint8_t buffer[MAX_MSG_SIZE];
    size_t msg_len;
    do
    {
        msg_len = sizeof(buffer) - 1;
        memset(buf, 0, sizeof(buffer));
        ret = mbedtls_ssl_read(&ssl, buffer, msg_len);

        if (ret == MBEDTLS_ERR_SSL_WANT_READ || ret == MBEDTLS_ERR_SSL_WANT_WRITE)
            continue;

        if (ret <= 0)
        {
            switch (ret)
            {
            case MBEDTLS_ERR_SSL_PEER_CLOSE_NOTIFY:
                mbedtls_printf(" connection was closed gracefully\n");
                break;

            case MBEDTLS_ERR_NET_CONN_RESET:
                mbedtls_printf(" connection was reset by peer\n");
                break;

            default:
                mbedtls_printf(" mbedtls_ssl_read returned -0x%x\n", -ret);
                break;
            }

            break;
        }

        msg_len = ret;
        if (ret > 0)
            break;
    } while (1);

    SecretShare *message;
    message = secret_share__unpack(NULL, msg_len, buffer);
    if (message == NULL)
    {
        fprintf(stderr, "Error unpacking incoming message\n");
    }

    // Display the message's fields
    box_out("[6] MAC Key Share + seed received from other player.\n");
    printf("Step 6 : Mac key and seed share received from player number %d : mackeyshare_2=%s", other_player_number, message->mackeyshare_2); // required field
    // printf("  seeds=%s\n", message->seeds);

    // // perform operations
    // char *temp = malloc(strlen(Seed) + 1); // +1 for the null terminator
    // if (temp == NULL)
    // {
    //     perror("malloc failed");
    //     return 1;
    // }
    // strcpy(temp, Seed);
    // Seed = addHex2(Seed, message->seeds);
    
    // Validate inputs before memcpy to prevent buffer overflow
    if (message == NULL || message->seeds == NULL || Seed == NULL)
    {
        fprintf(stderr, "Error: Invalid input parameters for Seed memcpy\n");
        goto exit;
    }
    
    // Call addHex2 and validate the result
    char *hex_result = addHex2(Seed, message->seeds);
    if (hex_result == NULL)
    {
        fprintf(stderr, "Error: addHex2 returned NULL\n");
        goto exit;
    }
    
    // Check the result length (use strnlen for safety to prevent over-read)
    size_t hex_result_len = strnlen(hex_result, KEY_LENGTH + 1);
    
    // Copy only the available bytes, up to KEY_LENGTH, to prevent buffer overflow
    size_t copy_len = (hex_result_len < KEY_LENGTH) ? hex_result_len : KEY_LENGTH;
    memcpy(Seed, hex_result, copy_len);
    
    // If the result was shorter than KEY_LENGTH, zero-pad the rest of Seed buffer
    if (hex_result_len < KEY_LENGTH)
    {
        memset(Seed + hex_result_len, 0, KEY_LENGTH - hex_result_len);
    }
    
    free(hex_result);  // Free the allocated memory from addHex2

    // Validate inputs before memcpy to prevent buffer overflow
    if (message == NULL || 
        message->mackeyshare_p == NULL || 
        message->mackeyshare_2 == NULL ||
        other_player_number < 0 || 
        other_player_number >= number_of_players ||
        Player_MAC_Keys_p[other_player_number] == NULL ||
        Player_MAC_Keys_2[other_player_number] == NULL)
    {
        fprintf(stderr, "Error: Invalid input parameters for memcpy\n");
        goto exit;
    }
    
    // Check source string lengths to prevent buffer overflow
    size_t mackeyshare_p_len = strnlen(message->mackeyshare_p, KEY_LENGTH + 1);
    size_t mackeyshare_2_len = strnlen(message->mackeyshare_2, KEY_LENGTH + 1);
    
    if (mackeyshare_p_len < KEY_LENGTH || mackeyshare_2_len < KEY_LENGTH)
    {
        fprintf(stderr, "Error: MAC key share length is too short (expected %d bytes)\n", KEY_LENGTH);
        goto exit;
    }
    
    memcpy(Player_MAC_Keys_p[other_player_number], message->mackeyshare_p, KEY_LENGTH);
    memcpy(Player_MAC_Keys_2[other_player_number], message->mackeyshare_2, KEY_LENGTH);
    // Free the unpacked message
    secret_share__free_unpacked(message, NULL);

    mbedtls_printf("  . Closing the connection...");

    while ((ret = mbedtls_ssl_close_notify(&ssl)) < 0)
    {
        if (ret != MBEDTLS_ERR_SSL_WANT_READ && ret != MBEDTLS_ERR_SSL_WANT_WRITE)
        {
            mbedtls_printf(" failed\n  ! mbedtls_ssl_close_notify returned %d\n\n", ret);
            goto exit;
        }
    }

    mbedtls_printf(" ok\n");
    other_player_number++;
    goto reset;


exit:
#ifdef MBEDTLS_ERROR_C
    if (ret != 0)
    {
        char error_buf[100];
        mbedtls_strerror(ret, error_buf, sizeof(error_buf));
        mbedtls_printf("Last error was: %d - %s\n\n", ret, error_buf);
    }
#endif
    fflush(stdout);
    if (ra_tls_attest_lib)
        dlclose(ra_tls_attest_lib);

    mbedtls_net_free(&client_fd);
    mbedtls_net_free(&listen_fd);

    mbedtls_x509_crt_free(&srvcert);
    mbedtls_pk_free(&pkey);
    mbedtls_ssl_free(&ssl);
    mbedtls_ssl_config_free(&conf);
    mbedtls_ctr_drbg_free(&ctr_drbg);
    mbedtls_entropy_free(&entropy);

    free(der_key);
    free(der_crt);

    return ret;
}
