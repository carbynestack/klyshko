/*
 * Copyright (c) 2025 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#include "vars.h"
#define MAC_KEY_SHARE_P_PATH "/etc/kii/secret-params/mac_key_share_p"
#define MAC_KEY_SHARE_2_PATH "/etc/kii/secret-params/mac_key_share_2"

void box_out(const char *str)
{
    // ANSI escape code for black text on green background
    mbedtls_printf("\033[30;42m"); // 30: black text, 42: green background
    mbedtls_printf("%s", str);     // Print the string
    mbedtls_printf("\033[0m\n");   // Reset to default colors
}

/* RA-TLS: on client, only need to register ra_tls_verify_callback_extended_der() for cert
 * verification. */
int (*ra_tls_verify_callback_extended_der_f)(uint8_t *der_crt, size_t der_crt_size,
                                             struct ra_tls_verify_callback_results *results);

/* RA-TLS: if specified in command-line options, use our own callback to verify SGX measurements */
void (*ra_tls_set_measurement_callback_f)(int (*f_cb)(const char *mrenclave, const char *mrsigner,
                                                      const char *isv_prod_id,
                                                      const char *isv_svn));

static void my_debug(void *ctx, int level, const char *file, int line, const char *str)
{
    ((void)level);

    mbedtls_fprintf((FILE *)ctx, "%s:%04d: %s\n", file, line, str);
    fflush((FILE *)ctx);
}

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

void read_file(const char *file_path, char **buffer)
{
    FILE *file = fopen(file_path, "r");
    if (!file)
    {
        fprintf(stderr, "Error: Could not open file %s\n", file_path);
        exit(EXIT_FAILURE);
    }

    fseek(file, 0, SEEK_END);
    long file_size = ftell(file);
    if (file_size < 0)
    {
        fprintf(stderr, "Error: Failed to determine file size for %s\n", file_path);
        fclose(file);
        exit(EXIT_FAILURE);
    }
    rewind(file);

    *buffer = (char *)malloc(file_size + 1);
    if (!*buffer)
    {
        fprintf(stderr, "Error: Memory allocation failed for file buffer\n");
        fclose(file);
        exit(EXIT_FAILURE);
    }

    size_t bytes_read = fread(*buffer, 1, file_size, file);
    if (bytes_read != file_size)
    {
        fprintf(stderr,
                "Error: Could not read the full file %s (expected %ld bytes, got %zu bytes)\n",
                file_path, file_size, bytes_read);
        free(*buffer);
        fclose(file);
        exit(EXIT_FAILURE);
    }

    (*buffer)[file_size] = '\0'; // Null-terminate the buffer
    fclose(file);
}

int main(int argc, char **argv)
{
    int ret;
    size_t len;
    mbedtls_net_context server_fd;
    uint32_t flags;
    unsigned char buf[1024];
    const char *pers = "ssl_client1";

    char *error;
    void *ra_tls_verify_lib = NULL;
    ra_tls_verify_callback_extended_der_f = NULL;
    ra_tls_set_measurement_callback_f = NULL;
    struct ra_tls_verify_callback_results my_verify_callback_results = {0};

    mbedtls_entropy_context entropy;
    mbedtls_ctr_drbg_context ctr_drbg;
    mbedtls_ssl_context ssl;
    mbedtls_ssl_config conf;

    char *b_port = getenv("BASE_PORT");
    int base_port = b_port ? atoi(b_port) : 0;

    char *current_player_number = argv[5];
    int player_number_defined = current_player_number ? atoi(current_player_number) : 0;

    int server_port = base_port + player_number_defined;
    char server_port_str[6]; // Assuming port numbers are within 5 digits
    snprintf(server_port_str, sizeof(server_port_str), "%d", server_port);

#if defined(MBEDTLS_DEBUG_C)
    mbedtls_debug_set_threshold(DEBUG_LEVEL);
#endif

    mbedtls_net_init(&server_fd);
    mbedtls_ssl_init(&ssl);
    mbedtls_ssl_config_init(&conf);
    mbedtls_ctr_drbg_init(&ctr_drbg);
    mbedtls_entropy_init(&entropy);

    ra_tls_verify_lib = dlopen("libra_tls_verify_dcap_gramine.so", RTLD_LAZY);
    if (!ra_tls_verify_lib)
    {
        mbedtls_printf("%s\n", dlerror());
        mbedtls_printf(
            "User requested RA-TLS verification with DCAP inside SGX but cannot find "
            "lib\n");
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

    if (argc > 2 && ra_tls_verify_lib)
    {
        if (argc != 6)
        {
            mbedtls_printf(
                "USAGE: %s %s <expected mrenclave> <expected mrsigner>"
                " <expected isv_prod_id> <expected isv_svn>\n"
                "       (first two in hex, last two as decimal; set to 0 to ignore)\n",
                argv[1], argv[2]);
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

        if (!strcmp(argv[1], "0"))
        {
            mbedtls_printf("  - ignoring MRENCLAVE\n");
            g_verify_mrenclave = false;
        }
        else if (parse_hex(argv[1], g_expected_mrenclave, sizeof(g_expected_mrenclave)) < 0)
        {
            mbedtls_printf("Cannot parse MRENCLAVE!\n");
            return 1;
        }

        if (!strcmp(argv[2], "0"))
        {
            mbedtls_printf("  - ignoring MRSIGNER\n");
            g_verify_mrsigner = false;
        }
        else if (parse_hex(argv[2], g_expected_mrsigner, sizeof(g_expected_mrsigner)) < 0)
        {
            mbedtls_printf("Cannot parse MRSIGNER!\n");
            return 1;
        }

        if (!strcmp(argv[3], "0"))
        {
            mbedtls_printf("  - ignoring ISV_PROD_ID\n");
            g_verify_isv_prod_id = false;
        }
        else
        {
            errno = 0;
            uint16_t isv_prod_id = (uint16_t)strtoul(argv[3], NULL, 10);
            if (errno)
            {
                mbedtls_printf("Cannot parse ISV_PROD_ID!\n");
                return 1;
            }
            // Validate buffer size before memcpy to prevent buffer overflow
            size_t dest_size = sizeof(g_expected_isv_prod_id);
            size_t src_size = sizeof(isv_prod_id);
            if (dest_size >= src_size)
            {
                // Copy only the source size, which is guaranteed to fit in destination
                memcpy(g_expected_isv_prod_id, &isv_prod_id, src_size);
            }
            else
            {
                mbedtls_printf("Error: Destination buffer too small for ISV_PROD_ID\n");
                return 1;
            }
        }

        if (!strcmp(argv[4], "0"))
        {
            mbedtls_printf("  - ignoring ISV_SVN\n");
            g_verify_isv_svn = false;
        }
        else
        {
            errno = 0;
            uint16_t isv_svn = (uint16_t)strtoul(argv[4], NULL, 10);
            if (errno)
            {
                mbedtls_printf("Cannot parse ISV_SVN\n");
                return 1;
            }
            // Validate buffer size before memcpy to prevent buffer overflow
            size_t dest_size = sizeof(g_expected_isv_svn);
            size_t src_size = sizeof(isv_svn);
            if (dest_size >= src_size)
            {
                // Copy only the source size, which is guaranteed to fit in destination
                memcpy(g_expected_isv_svn, &isv_svn, src_size);
            }
            else
            {
                mbedtls_printf("Error: Destination buffer too small for ISV_SVN\n");
                return 1;
            }
        }
    }
    else if (ra_tls_verify_lib)
    {
        mbedtls_printf(
            "[ using default SGX-measurement verification callback"
            " (via RA_TLS_* environment variables) ]\n");
        (*ra_tls_set_measurement_callback_f)(NULL); /* just to test RA-TLS code */
    }
    else
    {
        mbedtls_printf("[ using normal TLS flows ]\n");
    }

    mbedtls_printf("\n  . Seeding the random number generator...");
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

    mbedtls_printf("  . Connecting to tcp/%s/%s...", SERVER_NAME, server_port_str);
    fflush(stdout);

    while (1)
    {
        ret = mbedtls_net_connect(&server_fd, SERVER_NAME, server_port_str, MBEDTLS_NET_PROTO_TCP);
        if (ret == 0)
            break;
        else
        {
            mbedtls_printf(" failed\n  ! mbedtls_net_connect returned %d\n\n", ret);
            // mbedtls_printf("Retrying in %d seconds...\n", RETRY_DELAY);

            mbedtls_net_free(&server_fd);
            mbedtls_net_init(&server_fd);
            sleep(RETRY_DELAY);
        }
    }
    // if (ret != 0) {
    //     mbedtls_printf(" failed\n  ! mbedtls_net_connect returned %d\n\n", ret);
    //     goto exit;
    // }

    mbedtls_printf(" ok\n");

    mbedtls_printf("  . Setting up the SSL/TLS structure...");
    fflush(stdout);

    ret = mbedtls_ssl_config_defaults(&conf, MBEDTLS_SSL_IS_CLIENT, MBEDTLS_SSL_TRANSPORT_STREAM,
                                      MBEDTLS_SSL_PRESET_DEFAULT);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_config_defaults returned %d\n\n", ret);
        goto exit;
    }

    // mbedtls_printf(" ok\n");

    fflush(stdout);

    mbedtls_ssl_conf_authmode(&conf, MBEDTLS_SSL_VERIFY_OPTIONAL);
    mbedtls_printf(" ok\n");

    if (ra_tls_verify_lib)
    {
        /* use RA-TLS verification callback; this will overwrite CA chain set up above */
        mbedtls_printf("  . Installing RA-TLS callback ...");
        mbedtls_ssl_conf_verify(&conf, &my_verify_callback, &my_verify_callback_results);
        mbedtls_printf(" ok\n");
    }

    mbedtls_ssl_conf_rng(&conf, mbedtls_ctr_drbg_random, &ctr_drbg);
    mbedtls_ssl_conf_dbg(&conf, my_debug, stdout);

    ret = mbedtls_ssl_setup(&ssl, &conf);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_setup returned %d\n\n", ret);
        goto exit;
    }

    ret = mbedtls_ssl_set_hostname(&ssl, SERVER_NAME);
    if (ret != 0)
    {
        mbedtls_printf(" failed\n  ! mbedtls_ssl_set_hostname returned %d\n\n", ret);
        goto exit;
    }

    mbedtls_ssl_set_bio(&ssl, &server_fd, mbedtls_net_send, mbedtls_net_recv, NULL);

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
                my_verify_callback_results.attestation_scheme, my_verify_callback_results.err_loc);
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

    mbedtls_printf("  . Verifying peer X.509 certificate...");

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

        mbedtls_printf(" Step 2 local attestation of spawned TEE is successful\n");
        box_out("[2] Local attestation complete.");
    }

    mbedtls_printf(" ok\n");

    // PROTO BUFF STARTING
    // code for packing the macshares and sending over the TLS dconnection again to the server
    SecretShare message = SECRET_SHARE__INIT;
    char *macKeyShare_p = NULL;
    char *macKeyShare_2 = NULL;

    // Read macKeyShare_p
    read_file(MAC_KEY_SHARE_P_PATH, &macKeyShare_p);

    // Read macKeyShare_2
    read_file(MAC_KEY_SHARE_2_PATH, &macKeyShare_2);
    message.mackeyshare_p = macKeyShare_p;
    message.mackeyshare_2 = macKeyShare_2;
    unsigned length = secret_share__get_packed_size(&message);

    if (length == 0)
    {
        fprintf(stderr, "packing or serialization error");
    }
    void *buffer = malloc(length);
    if (!buffer)
    {
        fprintf(stderr, "Memory allocation error\n");
    }

    secret_share__pack(&message, buffer);
    fprintf(stderr, "Writing %d serialized bytes\n", length);
    while ((ret = mbedtls_ssl_write(&ssl, buffer, length)) <= 0)
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
    // EOC for sending

    length = ret;
    mbedtls_printf(" Step 3 MAC key shares shared to TEE\n");
    box_out("[3] MAC Key Shares received.");
    // PROTO BUFF ENDING

    while ((ret = mbedtls_ssl_close_notify(&ssl)) < 0)
    {
        if (ret != MBEDTLS_ERR_SSL_WANT_READ && ret != MBEDTLS_ERR_SSL_WANT_WRITE)
        {
            mbedtls_printf(" failed\n  ! mbedtls_ssl_close_notify returned %d\n\n", ret);
            goto exit;
        }
    }

exit:
#ifdef MBEDTLS_ERROR_C
    if (ret != 0)
    {
        char error_buf[100];
        mbedtls_strerror(ret, error_buf, sizeof(error_buf));
        mbedtls_printf("Last error was: %d - %s\n\n", ret, error_buf);
    }
#endif

    if (ra_tls_verify_lib)
        dlclose(ra_tls_verify_lib);

    mbedtls_net_free(&server_fd);

    mbedtls_ssl_free(&ssl);
    mbedtls_ssl_config_free(&conf);
    mbedtls_ctr_drbg_free(&ctr_drbg);
    mbedtls_entropy_free(&entropy);

    return ret;
}
