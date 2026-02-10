#!/usr/bin/env bash
#
# Copyright (c) 2022-2026 - for information on the respective copyright owner
# see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
#
# SPDX-License-Identifier: Apache-2.0
#

# Fail, if any command fails
set -e

mkdir -p ~/.cs
cat <<EOF > cs-config
{
  "prime" : 0,
  "r" : 0,
  "rinv" : 0,
  "noSslValidation" : true,
  "trustedCertificates" : [ ],
  "providers" : [ {
    "amphoraServiceUrl" : "http://ignore",
    "castorServiceUrl" : "http://cs-castor:10100",
    "ephemeralServiceUrl" : "http://ignore",
    "id" : 1,
    "baseUrl" : "http://ignore"
  }, {
    "amphoraServiceUrl" : "http://ignore",
    "castorServiceUrl" : "http://ignore",
    "ephemeralServiceUrl" : "http://ignore",
    "id" : 2,
    "baseUrl" : "http://ignore"
  } ]
}
EOF

# Derive a deterministic chunk UUID from a job ID and a piece index.
#
# Implements UUID v5 (SHA-1 based) using the RFC 4122 URL namespace
# (6ba7b811-9dad-11d1-80b4-00c04fd430c8) and the name string "<jobID>:<piece>".
# This matches DeriveChunkID() in the operator's chunk_utils.go exactly.
#
# Arguments:
#   $1 - job UUID string (e.g. "58301070-9b4c-4d38-ae89-d7aa989720a0")
#   $2 - piece index (integer)
# Output:
#   UUID string written to stdout
derive_chunk_id() {
  local job_id="$1"
  local piece="$2"

  # UUID v5 requires a namespace UUID as a 16-byte binary salt fed into SHA-1
  # together with the name.  We use the RFC 4122 "URL" namespace
  # (6ba7b811-9dad-11d1-80b4-00c04fd430c8), a well-known constant defined in
  # RFC 4122 Appendix C and also available in Go as uuid.NameSpaceURL.
  #
  # The hyphens are stripped because SHA-1 operates on raw bytes, not the
  # human-readable UUID string representation.  Keeping them would produce a
  # 20-byte input instead of the required 16, yielding a different hash than
  # Go's uuid.NewSHA1 — breaking cross-VCP correlation.
  #
  # The value itself is hardcoded because it is part of the algorithm
  # definition, not a runtime parameter: its sole purpose is to namespace the
  # derivation so that chunk IDs cannot collide with UUID v5s produced in any
  # other context.  The operator's chunk_utils.go uses the identical constant.
  # Name format matches DeriveChunkID() in chunk_utils.go: "<jobID>:<piece>".
  local name="${job_id}:${piece}"

  # Feed (namespace_bytes || name_bytes) into SHA-1 via a temp file.
  # A temp file is used instead of a pipe or command substitution to avoid
  # shell stripping of trailing newline bytes from binary data.
  local tmp
  tmp=$(mktemp)
  # Write the 16 binary bytes of the URL namespace UUID as a literal printf
  # escape sequence.  A literal format string (no variable) avoids SC2059
  # and removes the sed dependency.
  # 6ba7b811-9dad-11d1-80b4-00c04fd430c8 → \x6b\xa7\xb8\x11\x9d\xad\x11\xd1\x80\xb4\x00\xc0\x4f\xd4\x30\xc8
  printf '\x6b\xa7\xb8\x11\x9d\xad\x11\xd1\x80\xb4\x00\xc0\x4f\xd4\x30\xc8' >  "$tmp"
  printf '%s' "$name"                                                          >> "$tmp"
  local hash
  hash=$(sha1sum "$tmp" | cut -c1-40)
  rm -f "$tmp"

  # RFC 4122 §4.3: overwrite four bits of the third group with the version (5).
  # Mask out the top nibble with 0x0fff, then OR in 0x5000.
  local p3
  p3=$(printf '%04x' $(( (16#${hash:12:4} & 0x0fff) | 0x5000 )))

  # RFC 4122 §4.1.1: overwrite the top two bits of clock_seq_hi with 0b10
  # (the RFC 4122 variant).  Mask out those bits with 0x3f, then OR in 0x80.
  local cs_hi
  cs_hi=$(printf '%02x' $(( (16#${hash:16:2} & 0x3f) | 0x80 )))

  # Assemble the standard 8-4-4-4-12 UUID string from the modified hash bytes.
  printf '%s-%s-%s-%s%s-%s\n' \
    "${hash:0:8}" "${hash:8:4}" "$p3" "$cs_hi" "${hash:18:2}" "${hash:20:12}"
}

# Determine the number of upload chunks and the byte layout of the tuple file.
#
# The tuple file format is:
#   [ octetStream header (8-byte LE length prefix + content) ][ raw tuple data ]
#
# The header length prefix is read directly from the file so that no element
# size or field type needs to be hardcoded.
#
# Required environment variables:
#   KII_TUPLE_FILE       - path to the tuple file written by the generator
#   KII_TUPLES_PER_JOB   - total number of tuples in the file
#   KII_MAX_UPLOAD_TUPLES - maximum tuples per Castor upload chunk
#   KII_JOB_ID           - job UUID (used as chunk ID when there is only one chunk)
#   KII_TUPLE_TYPE       - tuple type string (passed through to the CLI)

FILE_SIZE=$(stat -c %s "${KII_TUPLE_FILE}")

# Read the 4-byte little-endian length of the octetStream header content.
# The header occupies bytes 0-7 (8-byte prefix) + <length> bytes of content.
HEADER_CONTENT_LEN=$(od -An -N4 -tu4 "${KII_TUPLE_FILE}" | tr -d ' \t\n')
HEADER_SIZE=$(( 8 + HEADER_CONTENT_LEN ))

DATA_SIZE=$(( FILE_SIZE - HEADER_SIZE ))
BYTES_PER_TUPLE=$(( DATA_SIZE / KII_TUPLES_PER_JOB ))

N_CHUNKS=$(( (KII_TUPLES_PER_JOB + KII_MAX_UPLOAD_TUPLES - 1) / KII_MAX_UPLOAD_TUPLES ))

if [ "${N_CHUNKS}" -eq 1 ]; then
  # Single chunk: use the original job ID directly (backward compatible).
  java -jar cs.jar --config-file cs-config castor upload-tuple \
    -f "${KII_TUPLE_FILE}" \
    -t "${KII_TUPLE_TYPE}" \
    -i "${KII_JOB_ID}" 1
else
  # Multiple chunks: split the tuple data and upload each piece separately.
  # Each piece file contains the original header followed by the piece's tuple data.
  TMPDIR=$(mktemp -d)
  trap 'rm -rf "${TMPDIR}"' EXIT

  for (( piece=0; piece<N_CHUNKS; piece++ )); do
    START_TUPLE=$(( piece * KII_MAX_UPLOAD_TUPLES ))
    REMAINING=$(( KII_TUPLES_PER_JOB - START_TUPLE ))
    TUPLES_THIS_CHUNK=$(( REMAINING < KII_MAX_UPLOAD_TUPLES ? REMAINING : KII_MAX_UPLOAD_TUPLES ))

    PIECE_FILE="${TMPDIR}/tuples-${piece}"

    # Copy header (small, bs=1 is fine)
    dd if="${KII_TUPLE_FILE}" bs=1 count="${HEADER_SIZE}" of="${PIECE_FILE}" status=none

    # Append the tuple data slice for this piece.
    # tail -c +N is 1-indexed; skip to byte (HEADER_SIZE + START_TUPLE*BYTES_PER_TUPLE + 1).
    DATA_START=$(( HEADER_SIZE + 1 + START_TUPLE * BYTES_PER_TUPLE ))
    DATA_LEN=$(( TUPLES_THIS_CHUNK * BYTES_PER_TUPLE ))
    tail -c "+${DATA_START}" "${KII_TUPLE_FILE}" | head -c "${DATA_LEN}" >> "${PIECE_FILE}"

    CHUNK_ID=$(derive_chunk_id "${KII_JOB_ID}" "${piece}")

    java -jar cs.jar --config-file cs-config castor upload-tuple \
      -f "${PIECE_FILE}" \
      -t "${KII_TUPLE_TYPE}" \
      -i "${CHUNK_ID}" 1
  done
fi
