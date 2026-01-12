# Unit Tests

This directory contains unit tests for the `klyshko-mp-spdz-tee` package using the cmocka testing framework.

## Test Files

- **CRG_test.c** - Tests for `CRG.c` (tuple types, file I/O, MAC key shares)
- **server_test.c** - Tests for `server.c` (player verification, hex parsing, file operations)
- **client_test.c** - Tests for `client.c` (hex arithmetic, parsing, file operations)

## Prerequisites

Install cmocka development libraries:

```bash
# On Ubuntu/Debian
sudo apt-get install libcmocka-dev

# On Fedora/RHEL
sudo dnf install libcmocka-devel

# On macOS (with Homebrew)
brew install cmocka
```

## Building and Running Tests

From the `tests` directory:

```bash
cd tests

# Build CRG test executable (uses actual CRG.c functions with stubs for TEE dependencies)
make CRG_test

# Run CRG tests
./CRG_test

# Or use make test
make test

# Generate text coverage report for CRG.c
make coverage-text
# This shows line-by-line coverage in CRG.c.gcov

# Generate HTML coverage report (requires lcov)
make coverage-html
# Then open coverage/html/index.html in a browser

# Clean up
make clean
```

## CRG.c Testing Setup

The CRG test setup uses **stubbing** to test actual `CRG.c` functions without requiring SGX hardware:

- **`vars_stub.h`** - Stub header that replaces `vars.h` without mbedtls/TEE dependencies
- **`stubs.c`** - Stub implementations of TEE functions (`local_attestation`, `ssl_*_setup_and_handshake`, etc.)
- **`pkg/vars.h`** - Wrapper that redirects to `vars_stub.h` for testing

This allows testing the actual `CRG.c` code (not reimplementations) and generating real coverage reports.


## Test Coverage

### CRG.c Tests
1. **`getTupleType()`** - Tests all valid tuple types and invalid inputs
2. **`get_random_hex()`** - Tests random hex string generation with various lengths
3. **`writeFile()`** - Tests file writing functionality
4. **`read_file()`** - Tests file reading functionality
5. **`create_mac_key_shares()`** - Tests MAC key share file creation for multiple players

### server.c Tests
1. **`verify_player_details()`** - Tests player and job ID verification logic
2. **`addHex2()`** - Tests hexadecimal addition with overflow handling
3. **`parse_hex()`** - Tests hex string parsing with various inputs
4. **`file_read()`** - Tests file reading with error handling
5. **`box_out()`** - Tests formatted output function

### client.c Tests
1. **`addHex()`** - Tests hexadecimal addition operations
2. **`parse_hex()`** - Tests hex string parsing
3. **`file_read()`** - Tests file reading operations



