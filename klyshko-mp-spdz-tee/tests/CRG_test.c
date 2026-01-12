#define _GNU_SOURCE
#include <stdarg.h>
#include <stddef.h>
#include <setjmp.h>
#include <cmocka.h>
#include <string.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sys/stat.h>
#include <errno.h>
#include <sys/random.h>

// Include stub header to replace vars.h (avoids TEE/mbedtls dependencies)
#include "vars_stub.h"

// Define TupleType enum to match CRG.c (needed for function signatures)
typedef enum
{
    BIT_GFP,
    BIT_GF2N,
    INPUT_MASK_GFP,
    INPUT_MASK_GF2N,
    INVERSE_TUPLE_GFP,
    INVERSE_TUPLE_GF2N,
    SQUARE_TUPLE_GFP,
    SQUARE_TUPLE_GF2N,
    MULTIPLICATION_TRIPLE_GFP,
    MULTIPLICATION_TRIPLE_GF2N,
    TUPLE_TYPE_COUNT
} TupleType;

// Declare functions from CRG.c that we want to test
// These are the actual functions from CRG.c, not reimplementations
extern TupleType getTupleType(const char *tuple_type_str);
extern void get_random_hex(char *hex_str, int length);
extern void writeFile(const char *filename, const char *text);
extern void read_file(const char *file_path, char **buffer);
extern void create_mac_key_shares(int pc, int pn, char *Player_MAC_Keys_p[], char *Player_MAC_Keys_2[]);

// ==================== Test Cases ====================

// Test getTupleType with all valid tuple types
static void test_getTupleType_valid_types(void **state)
{
    (void)state;

    assert_int_equal(getTupleType("BIT_GFP"), BIT_GFP);
    assert_int_equal(getTupleType("BIT_GF2N"), BIT_GF2N);
    assert_int_equal(getTupleType("INPUT_MASK_GFP"), INPUT_MASK_GFP);
    assert_int_equal(getTupleType("INPUT_MASK_GF2N"), INPUT_MASK_GF2N);
    assert_int_equal(getTupleType("INVERSE_TUPLE_GFP"), INVERSE_TUPLE_GFP);
    assert_int_equal(getTupleType("INVERSE_TUPLE_GF2N"), INVERSE_TUPLE_GF2N);
    assert_int_equal(getTupleType("SQUARE_TUPLE_GFP"), SQUARE_TUPLE_GFP);
    assert_int_equal(getTupleType("SQUARE_TUPLE_GF2N"), SQUARE_TUPLE_GF2N);
    assert_int_equal(getTupleType("MULTIPLICATION_TRIPLE_GFP"), MULTIPLICATION_TRIPLE_GFP);
    assert_int_equal(getTupleType("MULTIPLICATION_TRIPLE_GF2N"), MULTIPLICATION_TRIPLE_GF2N);
}

// Test getTupleType with invalid types
static void test_getTupleType_invalid_types(void **state)
{
    (void)state;

    assert_int_equal(getTupleType("INVALID_TYPE"), TUPLE_TYPE_COUNT);
    assert_int_equal(getTupleType(""), TUPLE_TYPE_COUNT);
    assert_int_equal(getTupleType("BIT"), TUPLE_TYPE_COUNT);
    assert_int_equal(getTupleType("BIT_GF"), TUPLE_TYPE_COUNT);
    // Note: getTupleType doesn't check for NULL, so calling it with NULL would segfault
    // We skip testing NULL to avoid crashes
}

// Test get_random_hex
static void test_get_random_hex(void **state)
{
    (void)state;

    char hex_str[17];
    get_random_hex(hex_str, 16);

    // Verify length
    assert_int_equal(strlen(hex_str), 16);

    // Verify all characters are valid hex digits
    for (int i = 0; i < 16; i++)
    {
        assert_true((hex_str[i] >= '0' && hex_str[i] <= '9') ||
                    (hex_str[i] >= 'a' && hex_str[i] <= 'f'));
    }

    // Verify null termination
    assert_int_equal(hex_str[16], '\0');
}

// Test get_random_hex with different lengths
static void test_get_random_hex_different_lengths(void **state)
{
    (void)state;

    char hex_str[33];
    
    // Test with length 8
    get_random_hex(hex_str, 8);
    assert_int_equal(strlen(hex_str), 8);
    assert_int_equal(hex_str[8], '\0');

    // Test with length 32
    get_random_hex(hex_str, 32);
    assert_int_equal(strlen(hex_str), 32);
    assert_int_equal(hex_str[32], '\0');
}

// Test writeFile
static void test_writeFile(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/crg_test_write.txt";
    const char *test_content = "test content line 1\ntest content line 2";

    writeFile(tmpfile, test_content);

    // Verify file was created and contains correct content
    FILE *f = fopen(tmpfile, "r");
    assert_non_null(f);

    char buffer[256];
    size_t bytes_read = fread(buffer, 1, sizeof(buffer) - 1, f);
    buffer[bytes_read] = '\0';
    fclose(f);

    assert_string_equal(buffer, test_content);

    // Cleanup
    unlink(tmpfile);
}

// Test writeFile with empty content
static void test_writeFile_empty_content(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/crg_test_write_empty.txt";

    writeFile(tmpfile, "");

    FILE *f = fopen(tmpfile, "r");
    assert_non_null(f);

    char buffer[256];
    size_t bytes_read = fread(buffer, 1, sizeof(buffer) - 1, f);
    buffer[bytes_read] = '\0';
    fclose(f);

    assert_int_equal(strlen(buffer), 0);

    unlink(tmpfile);
}

// Test read_file
static void test_read_file(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/crg_test_read.txt";
    const char *test_content = "hello world\nthis is a test";

    // Create test file
    FILE *f = fopen(tmpfile, "w");
    assert_non_null(f);
    fprintf(f, "%s", test_content);
    fclose(f);

    // Read file
    char *buffer = NULL;
    read_file(tmpfile, &buffer);

    assert_non_null(buffer);
    assert_string_equal(buffer, test_content);

    free(buffer);
    unlink(tmpfile);
}

// Test read_file with empty file
static void test_read_file_empty(void **state)
{
    (void)state;

    const char *tmpfile = "/tmp/crg_test_read_empty.txt";

    // Create empty file
    FILE *f = fopen(tmpfile, "w");
    assert_non_null(f);
    fclose(f);

    char *buffer = NULL;
    read_file(tmpfile, &buffer);

    assert_non_null(buffer);
    assert_int_equal(strlen(buffer), 0);

    free(buffer);
    unlink(tmpfile);
}

// Test create_mac_key_shares
static void test_create_mac_key_shares(void **state)
{
    (void)state;

    // Use a temporary directory for testing
    char original_cwd[1024];
    getcwd(original_cwd, sizeof(original_cwd));

    // Create test directory
    char test_dir[] = "/tmp/crg_test_player_data_XXXXXX";
    char *test_dir_actual = mkdtemp(test_dir);
    assert_non_null(test_dir_actual);

    chdir(test_dir_actual);

    // Allocate MAC keys
    int player_count = 2;
    char **keys_p = (char **)malloc(player_count * sizeof(char *));
    char **keys_2 = (char **)malloc(player_count * sizeof(char *));

    for (int i = 0; i < player_count; i++)
    {
        keys_p[i] = (char *)malloc(KEY_LENGTH * sizeof(char));
        keys_2[i] = (char *)malloc(KEY_LENGTH * sizeof(char));
        snprintf(keys_p[i], KEY_LENGTH, "test_key_p_%d", i);
        snprintf(keys_2[i], KEY_LENGTH, "test_key_2_%d", i);
    }

    // Create MAC key shares
    create_mac_key_shares(player_count, 0, keys_p, keys_2);

    // Verify files were created for field "p"
    struct stat st;
    char expected_file_p0[256];
    snprintf(expected_file_p0, sizeof(expected_file_p0),
             "Player-Data/%d-p-128/Player-MAC-Keys-p-P0", player_count);
    assert_int_equal(stat(expected_file_p0, &st), 0);

    char expected_file_p1[256];
    snprintf(expected_file_p1, sizeof(expected_file_p1),
             "Player-Data/%d-p-128/Player-MAC-Keys-p-P1", player_count);
    assert_int_equal(stat(expected_file_p1, &st), 0);

    // Verify files were created for field "2"
    char expected_file_2_0[256];
    snprintf(expected_file_2_0, sizeof(expected_file_2_0),
             "Player-Data/%d-2-40/Player-MAC-Keys-2-P0", player_count);
    assert_int_equal(stat(expected_file_2_0, &st), 0);

    char expected_file_2_1[256];
    snprintf(expected_file_2_1, sizeof(expected_file_2_1),
             "Player-Data/%d-2-40/Player-MAC-Keys-2-P1", player_count);
    assert_int_equal(stat(expected_file_2_1, &st), 0);

    // Verify file contents
    char *buffer = NULL;
    read_file(expected_file_p0, &buffer);
    assert_non_null(buffer);
    assert_string_equal(buffer, "2 test_key_p_0");
    free(buffer);

    read_file(expected_file_2_0, &buffer);
    assert_non_null(buffer);
    assert_string_equal(buffer, "2 test_key_2_0");
    free(buffer);

    // Cleanup
    for (int i = 0; i < player_count; i++)
    {
        free(keys_p[i]);
        free(keys_2[i]);
    }
    free(keys_p);
    free(keys_2);

    chdir(original_cwd);
    system("rm -rf /tmp/crg_test_player_data_*");
}

// Test create_mac_key_shares with 3 players
static void test_create_mac_key_shares_three_players(void **state)
{
    (void)state;

    char original_cwd[1024];
    getcwd(original_cwd, sizeof(original_cwd));

    char test_dir[] = "/tmp/crg_test_player_data_3_XXXXXX";
    char *test_dir_actual = mkdtemp(test_dir);
    assert_non_null(test_dir_actual);

    chdir(test_dir_actual);

    int player_count = 3;
    char **keys_p = (char **)malloc(player_count * sizeof(char *));
    char **keys_2 = (char **)malloc(player_count * sizeof(char *));

    for (int i = 0; i < player_count; i++)
    {
        keys_p[i] = (char *)malloc(KEY_LENGTH * sizeof(char));
        keys_2[i] = (char *)malloc(KEY_LENGTH * sizeof(char));
        snprintf(keys_p[i], KEY_LENGTH, "key_p_%d", i);
        snprintf(keys_2[i], KEY_LENGTH, "key_2_%d", i);
    }

    create_mac_key_shares(player_count, 0, keys_p, keys_2);

    // Verify all 3 players have files
    struct stat st;
    for (int i = 0; i < player_count; i++)
    {
        char file_p[256];
        snprintf(file_p, sizeof(file_p),
                 "Player-Data/%d-p-128/Player-MAC-Keys-p-P%d", player_count, i);
        assert_int_equal(stat(file_p, &st), 0);

        char file_2[256];
        snprintf(file_2, sizeof(file_2),
                 "Player-Data/%d-2-40/Player-MAC-Keys-2-P%d", player_count, i);
        assert_int_equal(stat(file_2, &st), 0);
    }

    // Cleanup
    for (int i = 0; i < player_count; i++)
    {
        free(keys_p[i]);
        free(keys_2[i]);
    }
    free(keys_p);
    free(keys_2);

    chdir(original_cwd);
    system("rm -rf /tmp/crg_test_player_data_3_*");
}

// ==================== Test Runner ====================

int main(void)
{
    const struct CMUnitTest tests[] = {
        // getTupleType tests
        cmocka_unit_test(test_getTupleType_valid_types),
        cmocka_unit_test(test_getTupleType_invalid_types),

        // get_random_hex tests
        cmocka_unit_test(test_get_random_hex),
        cmocka_unit_test(test_get_random_hex_different_lengths),

        // writeFile tests
        cmocka_unit_test(test_writeFile),
        cmocka_unit_test(test_writeFile_empty_content),

        // read_file tests
        cmocka_unit_test(test_read_file),
        cmocka_unit_test(test_read_file_empty),

        // create_mac_key_shares tests
        cmocka_unit_test(test_create_mac_key_shares),
        cmocka_unit_test(test_create_mac_key_shares_three_players),
    };

    return cmocka_run_group_tests(tests, NULL, NULL);
}

