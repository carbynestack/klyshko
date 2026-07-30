/*
 * Copyright (c) 2026 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#include <stdarg.h>
#include <stddef.h>
#include <setjmp.h>
#include <cmocka.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <errno.h>
#include <ctype.h>
#include <stdint.h>

// Include parse_hex and file_read implementations from server.c
static int parse_hex(const char *hex, void *buffer, size_t buffer_size)
{
    size_t expected_len = buffer_size * 2;
    size_t len = 0;
    while (len <= expected_len && hex[len] != '\0')
        len++;
    if (len != expected_len)
        return -1;

    for (size_t i = 0; i < buffer_size; i++)
    {
        if (!isxdigit(hex[i * 2]) || !isxdigit(hex[i * 2 + 1]))
            return -1;
        sscanf(hex + i * 2, "%02hhx", &((uint8_t *)buffer)[i]);
    }
    return 0;
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

// Include verify_player_details and addHex2 from server.c
int verify_player_details(char *kii_job_id, int received_other_player_number,
                          char *kii_job_id_defined, int other_player_number)
{
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

void box_out(const char *str)
{
    // ANSI escape code for black text on green background
    printf("\033[30;42m"); // 30: black text, 42: green background
    printf("%s", str);     // Print the string
    printf("\033[0m\n");   // Reset to default colors
}

// ==================== Test Cases ====================

// Test verify_player_details - success case
static void test_verify_player_details_success(void **state)
{
    (void)state;

    char *job_id = "test-job-123";
    char *defined_job_id = "test-job-123";
    int received_player = 1;
    int expected_player = 1;

    assert_int_equal(verify_player_details(job_id, received_player, defined_job_id, expected_player), 0);
}

// Test verify_player_details - wrong job id
static void test_verify_player_details_wrong_job_id(void **state)
{
    (void)state;

    char *job_id = "test-job-123";
    char *defined_job_id = "test-job-456";
    int received_player = 1;
    int expected_player = 1;

    assert_int_equal(verify_player_details(job_id, received_player, defined_job_id, expected_player), -1);
}

// Test verify_player_details - wrong player number
static void test_verify_player_details_wrong_player(void **state)
{
    (void)state;

    char *job_id = "test-job-123";
    char *defined_job_id = "test-job-123";
    int received_player = 1;
    int expected_player = 2;

    assert_int_equal(verify_player_details(job_id, received_player, defined_job_id, expected_player), -1);
}

// Test verify_player_details - both wrong
static void test_verify_player_details_both_wrong(void **state)
{
    (void)state;

    char *job_id = "test-job-123";
    char *defined_job_id = "test-job-456";
    int received_player = 1;
    int expected_player = 2;

    assert_int_equal(verify_player_details(job_id, received_player, defined_job_id, expected_player), -1);
}

// Test verify_player_details - empty strings
static void test_verify_player_details_empty_strings(void **state)
{
    (void)state;

    char *job_id = "";
    char *defined_job_id = "";
    int received_player = 0;
    int expected_player = 0;

    assert_int_equal(verify_player_details(job_id, received_player, defined_job_id, expected_player), 0);
}

// Test parse_hex - valid hex string
static void test_parse_hex_valid(void **state)
{
    (void)state;

    uint8_t buffer[4];
    assert_int_equal(parse_hex("0A1B2C3D", buffer, 4), 0);
    assert_int_equal(buffer[0], 0x0A);
    assert_int_equal(buffer[1], 0x1B);
    assert_int_equal(buffer[2], 0x2C);
    assert_int_equal(buffer[3], 0x3D);
}

// Test parse_hex - wrong length
static void test_parse_hex_wrong_length(void **state)
{
    (void)state;

    uint8_t buffer[4];
    assert_int_equal(parse_hex("0A1B2C", buffer, 4), -1);  // Too short
    assert_int_equal(parse_hex("0A1B2C3D4E", buffer, 4), -1); // Too long
}

// Test parse_hex - invalid characters
static void test_parse_hex_invalid_chars(void **state)
{
    (void)state;

    uint8_t buffer[4];
    assert_int_equal(parse_hex("ZZ001122", buffer, 4), -1);
    assert_int_equal(parse_hex("0A1B2C3G", buffer, 4), -1);
    assert_int_equal(parse_hex("0A1B2C3g", buffer, 4), -1);
}

// Test parse_hex - empty string
static void test_parse_hex_empty(void **state)
{
    (void)state;

    uint8_t buffer[1];
    assert_int_equal(parse_hex("", buffer, 1), -1);
}

// Test parse_hex - single byte
static void test_parse_hex_single_byte(void **state)
{
    (void)state;

    uint8_t buffer[1];
    assert_int_equal(parse_hex("FF", buffer, 1), 0);
    assert_int_equal(buffer[0], 0xFF);
}

// Test parse_hex - lowercase hex
static void test_parse_hex_lowercase(void **state)
{
    (void)state;

    uint8_t buffer[2];
    assert_int_equal(parse_hex("abcd", buffer, 2), 0);
    assert_int_equal(buffer[0], 0xAB);
    assert_int_equal(buffer[1], 0xCD);
}

// Test addHex2 - basic addition
static void test_addHex2_basic(void **state)
{
    (void)state;

    char *result = addHex2("0000000000000001", "0000000000000002");
    assert_non_null(result);
    assert_string_equal(result, "0000000000000003");
    free(result);
}

// Test addHex2 - larger numbers
static void test_addHex2_large(void **state)
{
    (void)state;

    char *result = addHex2("00000000000000FF", "0000000000000001");
    assert_non_null(result);
    assert_string_equal(result, "0000000000000100");
    free(result);
}

// Test addHex2 - overflow handling
static void test_addHex2_overflow(void **state)
{
    (void)state;

    char *result = addHex2("FFFFFFFFFFFFFFFF", "0000000000000001");
    assert_non_null(result);
    // Should wrap around
    assert_string_equal(result, "0000000000000000");
    free(result);
}

// Test addHex2 - zero addition
static void test_addHex2_zero(void **state)
{
    (void)state;

    char *result = addHex2("0000000000000000", "0000000000000000");
    assert_non_null(result);
    assert_string_equal(result, "0000000000000000");
    free(result);
}

// Note: addHex2 doesn't check for NULL inputs, so we skip testing NULL cases
// In production code, you might want to add NULL checks

// Test file_read - valid file
static void test_file_read_valid(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/server_test_file.txt";
    static const char test_content[] = "test content for file_read";

    // Create test file
    FILE *f = fopen(tmpfile, "w");
    assert_non_null(f);
    fprintf(f, "%s", test_content);
    fclose(f);

    char buffer[256] = {0};
    ssize_t bytes = file_read(tmpfile, buffer, sizeof(buffer) - 1);

    assert_int_equal(bytes, (ssize_t)(sizeof(test_content) - 1));
    assert_string_equal(buffer, test_content);

    unlink(tmpfile);
}

// Test file_read - file not found
static void test_file_read_not_found(void **state)
{
    (void)state;

    char buffer[256];
    ssize_t bytes = file_read("/tmp/nonexistent_file_12345.txt", buffer, sizeof(buffer));

    assert_true(bytes < 0); // Should return negative error code
}

// Test file_read - empty file
static void test_file_read_empty(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/server_test_empty.txt";

    // Create empty file
    FILE *f = fopen(tmpfile, "w");
    assert_non_null(f);
    fclose(f);

    char buffer[256] = {0};
    ssize_t bytes = file_read(tmpfile, buffer, sizeof(buffer));

    // Empty file should return 0 or negative
    assert_true(bytes <= 0);

    unlink(tmpfile);
}

// Test file_read - small buffer
static void test_file_read_small_buffer(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/server_test_small.txt";
    const char *test_content = "12345";

    FILE *f = fopen(tmpfile, "w");
    assert_non_null(f);
    fprintf(f, "%s", test_content);
    fclose(f);

    char buffer[3] = {0};
    ssize_t bytes = file_read(tmpfile, buffer, sizeof(buffer) - 1);

    assert_int_equal(bytes, 2); // Should read 2 bytes (buffer size - 1)
    assert_string_equal(buffer, "12");

    unlink(tmpfile);
}

// Test box_out - basic functionality (just verify it doesn't crash)
static void test_box_out_basic(void **state)
{
    (void)state;

    // box_out should not crash
    box_out("Test message");
    box_out("");
    box_out("Another test");
}

// ==================== Test Runner ====================

int main(void)
{
    const struct CMUnitTest tests[] = {
        // verify_player_details tests
        cmocka_unit_test(test_verify_player_details_success),
        cmocka_unit_test(test_verify_player_details_wrong_job_id),
        cmocka_unit_test(test_verify_player_details_wrong_player),
        cmocka_unit_test(test_verify_player_details_both_wrong),
        cmocka_unit_test(test_verify_player_details_empty_strings),

        // parse_hex tests
        cmocka_unit_test(test_parse_hex_valid),
        cmocka_unit_test(test_parse_hex_wrong_length),
        cmocka_unit_test(test_parse_hex_invalid_chars),
        cmocka_unit_test(test_parse_hex_empty),
        cmocka_unit_test(test_parse_hex_single_byte),
        cmocka_unit_test(test_parse_hex_lowercase),

        // addHex2 tests
        cmocka_unit_test(test_addHex2_basic),
        cmocka_unit_test(test_addHex2_large),
        cmocka_unit_test(test_addHex2_overflow),
        cmocka_unit_test(test_addHex2_zero),

        // file_read tests
        cmocka_unit_test(test_file_read_valid),
        cmocka_unit_test(test_file_read_not_found),
        cmocka_unit_test(test_file_read_empty),
        cmocka_unit_test(test_file_read_small_buffer),

        // box_out tests
        cmocka_unit_test(test_box_out_basic),
    };

    return cmocka_run_group_tests(tests, NULL, NULL);
}

