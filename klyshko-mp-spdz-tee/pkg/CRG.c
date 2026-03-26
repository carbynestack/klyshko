/*
 * Copyright (c) 2026 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#define EXTERN
#include "vars.h"

char **kii_endpoints;
#include <fcntl.h>
#include <sys/stat.h>

// Define an enumeration for tuple types
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
    TUPLE_TYPE_COUNT // Count of types, useful for bounds checking
} TupleType;

// Arrays for command arguments and file paths corresponding to the tuple types
const char *arg1ByType[TUPLE_TYPE_COUNT] = {
    "--nbits",     // BIT_GFP
    "--nbits",     // BIT_GF2N
    "--ntriples",  // INPUT_MASK_GFP
    "--ntriples",  // INPUT_MASK_GF2N
    "--ninverses", // INVERSE_TUPLE_GFP
    "--ninverses", // INVERSE_TUPLE_GF2N
    "--nsquares",  // SQUARE_TUPLE_GFP
    "--nsquares",  // SQUARE_TUPLE_GF2N
    "--ntriples",  // MULTIPLICATION_TRIPLE_GFP
    "--ntriples"   // MULTIPLICATION_TRIPLE_GF2N
};

// arg2FormatByType array removed - replaced with explicit switch statement
// to prevent format string vulnerabilities (format strings are now hardcoded constants)

// tupleFileByType array removed - replaced with explicit switch statement
// to prevent format string vulnerabilities (format strings are now hardcoded constants)

// Helper function to convert string to TupleType enum
TupleType getTupleType(const char *tuple_type_str)
{
    if (strcmp(tuple_type_str, "BIT_GFP") == 0)
        return BIT_GFP;
    if (strcmp(tuple_type_str, "BIT_GF2N") == 0)
        return BIT_GF2N;
    if (strcmp(tuple_type_str, "INPUT_MASK_GFP") == 0)
        return INPUT_MASK_GFP;
    if (strcmp(tuple_type_str, "INPUT_MASK_GF2N") == 0)
        return INPUT_MASK_GF2N;
    if (strcmp(tuple_type_str, "INVERSE_TUPLE_GFP") == 0)
        return INVERSE_TUPLE_GFP;
    if (strcmp(tuple_type_str, "INVERSE_TUPLE_GF2N") == 0)
        return INVERSE_TUPLE_GF2N;
    if (strcmp(tuple_type_str, "SQUARE_TUPLE_GFP") == 0)
        return SQUARE_TUPLE_GFP;
    if (strcmp(tuple_type_str, "SQUARE_TUPLE_GF2N") == 0)
        return SQUARE_TUPLE_GF2N;
    if (strcmp(tuple_type_str, "MULTIPLICATION_TRIPLE_GFP") == 0)
        return MULTIPLICATION_TRIPLE_GFP;
    if (strcmp(tuple_type_str, "MULTIPLICATION_TRIPLE_GF2N") == 0)
        return MULTIPLICATION_TRIPLE_GF2N;

    // Default case, handle unknown types (could be an error handling mechanism)
    return TUPLE_TYPE_COUNT;
}

void get_random_hex(char *hex_str, int length)
{
    const char hex_chars[] = "0123456789abcdef"; 
    unsigned char random_bytes[length];

    // Get truly random bytes from the system
    if (getrandom(random_bytes, length, 0) == -1)
    {
        perror("getrandom");
        exit(EXIT_FAILURE);
    }

    for (int i = 0; i < length; i++)
    {
        hex_str[i] = hex_chars[random_bytes[i] % 16]; // Convert to hex
    }

    hex_str[length] = '\0'; // Null-terminate the string
}

void writeFile(const char *filename, const char *text)
{
    // Open the file for writing
    FILE *file = fopen(filename, "w");
    if (file == NULL)
    {
        perror("Error openin file while writing");
        exit(1); // Exit if there's an error opening the file
    }
    else
    {
        // perror("opened successfully");
    }

    fprintf(file, "%s", text);

    fclose(file);
}

void create_mac_key_shares(int pc, int pn, char *Player_MAC_Keys_p[], char *Player_MAC_Keys_2[])
{
    const char *fields[] = {"p", "2"};

    for (size_t i = 0; i < sizeof(fields) / sizeof(fields[0]); ++i)
    {
        const char *f = fields[i];
        const char *bit_width = (strcmp(f, "p") == 0) ? "128" : "40";

        char *folder = "Player-Data/";

        char folderPath[256];
        snprintf(folderPath, sizeof(folderPath), "%s%d-%s-%s", folder, pc, f, bit_width);

        printf("Providing parameters for field %s-%s in folder %s\n", f, bit_width, folder);

        // Write MAC key shares for all players
        for (int playerNumber = 0; playerNumber < pc; ++playerNumber)
        {
            char macKeyShareFile[256];
            snprintf(macKeyShareFile, sizeof(macKeyShareFile), "%s/Player-MAC-Keys-%s-P%d",
                     folderPath, f, playerNumber);

            char *macKeyShare;
            char file_path[256];
            if (f == "p")
            {
                macKeyShare = Player_MAC_Keys_p[playerNumber];
            }
            else
            {
                macKeyShare = Player_MAC_Keys_2[playerNumber];
            }

            char dataToWrite[256];

            snprintf(dataToWrite, sizeof(dataToWrite), "%d %s", pc, macKeyShare);
            writeFile(macKeyShareFile, dataToWrite);

        }
    }
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
    int other_player_number = 0;
    char *prime = NULL;
    read_file("/etc/kii/params/prime", &prime);
    char Seed[17];

    get_random_hex(Seed, 16);
    printf("Getting environment variables ...\n");
    const char *env_names[] = {"KII_TUPLES_PER_JOB", "KII_SHARED_FOLDER", "KII_TUPLE_FILE",
                               "KII_PLAYER_NUMBER", "KII_PLAYER_COUNT", "KII_JOB_ID",
                               "KII_TUPLE_TYPE", "BASE_PORT"};

    char *env_values[sizeof(env_names) / sizeof(env_names[0])];

    // Loop through each environment variable
    for (int i = 0; i < sizeof(env_names) / sizeof(env_names[0]); i++)
    {
        env_values[i] = getenv(env_names[i]);

        // Check if the environment variable exists and print the appropriate message
        if (env_values[i] == NULL)
        {
            fprintf(stderr, "Error: Environment variable %s not found.\n", env_names[i]);
        }
    }
    char *n = env_values[0];
    char *tuple_file = env_values[2];
    char *tuple_type_str = env_values[6];
    char *kii_job_id_str = env_values[5];    // KII_JOB_ID
    char *player_number_str = env_values[3]; // KII_PLAYER_NUMBER
    char *number_of_players_str = env_values[4];
    char *b_port = env_values[7];
    // Convert to integers
    kii_job_id_defined = kii_job_id_str; // Check for NULL
    player_number_defined = player_number_str ? atoi(player_number_str) : 0;
    number_of_players = number_of_players_str ? atoi(number_of_players_str) : 0;
    base_port = b_port ? atoi(b_port) : 0;
    // EOC for getting the env variables and storing them inside main fuction

    kii_endpoints = (char **)malloc(number_of_players * sizeof(char *));
    for (int i = 0; i < number_of_players; i++)
    {
        char env_kii_name[35]; // Buffer to hold the variable name
        snprintf(env_kii_name, sizeof(env_kii_name), "KII_PLAYER_ENDPOINT_%d", i);
        kii_endpoints[i] = getenv(env_kii_name);
        if (kii_endpoints[i] != NULL)
        {
            printf("Player %d endpoint: %s\n", i, kii_endpoints[i]);
        }
        else
        {
            printf("Environment variable %s is not set.\n", env_kii_name);
        }
    }


#if defined(MBEDTLS_DEBUG_C)
    mbedtls_debug_set_threshold(DEBUG_LEVEL);
#endif

    char **Player_MAC_Keys_p = (char **)malloc(number_of_players * sizeof(char *));
    char **Player_MAC_Keys_2 = (char **)malloc(number_of_players * sizeof(char *));

    // Allocate space for each player's key
    for (int i = 0; i < number_of_players; i++)
    {
        Player_MAC_Keys_p[i] = (char *)malloc(KEY_LENGTH * sizeof(char));
        Player_MAC_Keys_2[i] = (char *)malloc(KEY_LENGTH * sizeof(char));

        if (Player_MAC_Keys_p[i] == NULL || Player_MAC_Keys_2[i] == NULL)
        {
            printf("Memory allocation failed for player %d\n", i);
            exit(1);
        }
    }

    printf("Local attestation starts . . .\n");
    ret = local_attestation(Player_MAC_Keys_p, Player_MAC_Keys_2);
    if (ret != 0)
        exit(ret);
    printf("End of CRG.c local attestation\n");
    printf("Remote attestation starts..\n");
    if (player_number_defined > 0)
    {
        ret = ssl_server_setup_and_handshake(argv[1], argv[2], argv[3], argv[4], Player_MAC_Keys_p,
                                             Player_MAC_Keys_2, Seed);
        if (ret != 0)
            exit(ret);
    }
    if (player_number_defined < number_of_players - 1)
    {
        ret = ssl_client_setup_and_handshake(argv[1], argv[2], argv[3], argv[4], Player_MAC_Keys_p,
                                             Player_MAC_Keys_2, Seed);

        if (ret != 0)
            exit(ret);
    }

    printf("End of Remote attestation..\n");

    TupleType tuple_type = getTupleType(tuple_type_str);
    if (tuple_type == TUPLE_TYPE_COUNT)
    {
        fprintf(stderr, "Unknown tuple type: %s\n", tuple_type_str);
        return 1;
    }

    char arg2[256] = {0};
    // Use explicit format strings in switch statement to prevent format string vulnerabilities
    // All format strings are hardcoded constants, not user-controlled
    // Validate tuple_type is within bounds
    if (tuple_type >= TUPLE_TYPE_COUNT)
    {
        fprintf(stderr, "Error: Invalid tuple type index: %d\n", tuple_type);
        return 1;
    }
    
    // Validate n is not NULL before using in format strings
    if (n == NULL)
    {
        fprintf(stderr, "Error: KII_TUPLES_PER_JOB environment variable not set\n");
        return 1;
    }
    
    switch (tuple_type)
    {
        case BIT_GFP:
            snprintf(arg2, sizeof(arg2), "0,%s", n);
            break;
        case BIT_GF2N:
            snprintf(arg2, sizeof(arg2), "%s,0", n);
            break;
        case INPUT_MASK_GFP:
            snprintf(arg2, sizeof(arg2), "0,%d", atoi(n) / 3);
            break;
        case INPUT_MASK_GF2N:
            snprintf(arg2, sizeof(arg2), "%d,0", atoi(n) / 3);
            break;
        case INVERSE_TUPLE_GFP:
        case INVERSE_TUPLE_GF2N:
            snprintf(arg2, sizeof(arg2), "%s", n);
            break;
        case SQUARE_TUPLE_GFP:
            snprintf(arg2, sizeof(arg2), "0,%s", n);
            break;
        case SQUARE_TUPLE_GF2N:
            snprintf(arg2, sizeof(arg2), "%s,0", n);
            break;
        case MULTIPLICATION_TRIPLE_GFP:
            snprintf(arg2, sizeof(arg2), "0,%s", n);
            break;
        case MULTIPLICATION_TRIPLE_GF2N:
            snprintf(arg2, sizeof(arg2), "%s,0", n);
            break;
        default:
            fprintf(stderr, "Error: Unhandled tuple type: %d\n", tuple_type);
            return 1;
    }

    int player_count = atoi(number_of_players_str);
    int player_number = atoi(player_number_str);
    create_mac_key_shares(player_count, player_number, Player_MAC_Keys_p, Player_MAC_Keys_2);
    printf("Running Fake Offline as execvp process\n");
    box_out("[7] Running Fake Offline.\n");
    fflush(stdout);
    char *args[] = {
        "../Fake-Offline.x",
        "-d",
        "0",
        "--prime",
        prime,
        "--prngseed",
        Seed,
        arg1ByType[tuple_type], // The argument part 1 (e.g., --nbits)
        arg2,                   // The argument part 2 (e.g., 0,1000)
        number_of_players_str,  // Player count
        NULL                    // Terminate with NULL
    };

    
    // Execute ./Fake-Offline.x using execvp
    int length = sizeof(args) / sizeof(args[0]);

    // Join cmd array into a single command string
    char cmdString[512] = {0}; // Buffer to hold the concatenated cmd
    size_t cmdString_len = 0;  // Track current length to prevent buffer overflow
    for (int i = 0; i < length; ++i)
    {
        if (args[i] != NULL)
        {
            // Calculate remaining space in buffer (leave 1 byte for null terminator)
            size_t remaining = sizeof(cmdString) - cmdString_len - 1;
            
            // Get the length of the argument to append (safely with strnlen)
            size_t arg_len = strnlen(args[i], remaining);
            
            // Check if we have enough space for the argument
            // For first argument: need space for arg + null terminator
            // For subsequent arguments: need space for space + arg + null terminator
            size_t space_needed = (cmdString_len > 0) ? arg_len + 1 : arg_len; // +1 for space before arg if not first
            
            if (space_needed <= remaining)
            {
                // Use snprintf to safely append argument (safer than strncat)
                // For first argument: just append the argument
                // For subsequent arguments: append space + argument
                int written;
                if (cmdString_len > 0)
                {
                    // Not the first argument - add space before it
                    written = snprintf(cmdString + cmdString_len, remaining + 1, " %s", args[i]);
                }
                else
                {
                    // First argument - no leading space
                    written = snprintf(cmdString + cmdString_len, remaining + 1, "%s", args[i]);
                }
                
                // Check if snprintf succeeded (written >= 0) and didn't truncate (written < remaining + 1)
                if (written < 0)
                {
                    fprintf(stderr, "Error: snprintf failed while building command string\n");
                    break;
                }
                else if ((size_t)written >= remaining + 1)
                {
                    fprintf(stderr, "Error: Command string too long, cannot add argument: %s\n", args[i]);
                    break;
                }
                
                cmdString_len += written;  // Update length with actual bytes written
            }
            else
            {
                fprintf(stderr, "Error: Command string too long, cannot add argument: %s\n", args[i]);
                break;
            }
        }
    }

    char destination_path[1024] = {0};
    // Use explicit format strings in switch statement to prevent format string vulnerabilities
    // All format strings are hardcoded constants, not user-controlled
    switch (tuple_type)
    {
        case BIT_GFP:
            snprintf(destination_path, sizeof(destination_path), "%s-p-128/Bits-p-P%s", number_of_players_str, player_number_str);
            break;
        case BIT_GF2N:
            snprintf(destination_path, sizeof(destination_path), "%s-2-40/Bits-2-P%s", number_of_players_str, player_number_str);
            break;
        case INPUT_MASK_GFP:
            snprintf(destination_path, sizeof(destination_path), "%s-p-128/Triples-p-P%s", number_of_players_str, player_number_str);
            break;
        case INPUT_MASK_GF2N:
            snprintf(destination_path, sizeof(destination_path), "%s-2-40/Triples-2-P%s", number_of_players_str, player_number_str);
            break;
        case INVERSE_TUPLE_GFP:
            snprintf(destination_path, sizeof(destination_path), "%s-p-128/Inverses-p-P%s", number_of_players_str, player_number_str);
            break;
        case INVERSE_TUPLE_GF2N:
            snprintf(destination_path, sizeof(destination_path), "%s-2-40/Inverses-2-P%s", number_of_players_str, player_number_str);
            break;
        case SQUARE_TUPLE_GFP:
            snprintf(destination_path, sizeof(destination_path), "%s-p-128/Squares-p-P%s", number_of_players_str, player_number_str);
            break;
        case SQUARE_TUPLE_GF2N:
            snprintf(destination_path, sizeof(destination_path), "%s-2-40/Squares-2-P%s", number_of_players_str, player_number_str);
            break;
        case MULTIPLICATION_TRIPLE_GFP:
            snprintf(destination_path, sizeof(destination_path), "%s-p-128/Triples-p-P%s", number_of_players_str, player_number_str);
            break;
        case MULTIPLICATION_TRIPLE_GF2N:
            snprintf(destination_path, sizeof(destination_path), "%s-2-40/Triples-2-P%s", number_of_players_str, player_number_str);
            break;
        default:
            fprintf(stderr, "Error: Unhandled tuple type for destination path: %d\n", tuple_type);
            return 1;
    }

    // Construct the full command with the copy operation
    char fullCommand[1024] = {0}; // Buffer for the full command
    snprintf(fullCommand, sizeof(fullCommand), "%s&& cp Player-Data/%s %s", cmdString, destination_path, tuple_file);

    // Prepare the args for execvp
    char *cmd[] = {"/bin/bash", "-c", fullCommand, NULL};

    // Step 8: Execute ./Fake-Offline.x using execvp
    execvp("/bin/bash", cmd);
    // If execvp fails:
    perror("execvp failed");
    // ssl_client_setup_and_handshake(argv[1], argv[2], argv[3], argv[4]);

    // Free allocated memory
    for (int i = 0; i < number_of_players; i++)
    {
        free(Player_MAC_Keys_p[i]);
        free(Player_MAC_Keys_2[i]);
    }

    // free KII_endpoint
    free(kii_endpoints);
    kii_endpoints = NULL; // Set the pointer to NULL after freeing
}
