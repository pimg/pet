# Bloom Filter Exchange Format

This document specifies the binary format used to persist a Bloom filter and exchange it between systems. It defines the byte layout, how element positions are derived, and the rules writers and readers must follow. Any implementation that follows this document, in any language, can read filters written by another.

The key words **MUST**, **MUST NOT**, **SHOULD** and **MAY** are used as described in RFC 2119.

## Overview

A serialized filter is a single, self-delimiting byte sequence made of three parts:

```
+-------------------+------------------------------+-----------------+
| Header (44 bytes) | Bit array ((m + 7) / 8 bytes) | CRC-32C (4 bytes) |
+-------------------+------------------------------+-----------------+
```

The header contains everything needed to rebuild the filter: the number of bits `m`, the number of hash positions per element `k`, the hash algorithm and its seeds. Because `m` is stored in the header, a reader knows the exact total length after reading the first 44 bytes.

## Conventions

- All multi-byte integers are **unsigned and little-endian**.
- All fields have a fixed width. The format uses no delimiters, padding or alignment beyond what is listed in the layout.
- Offsets and sizes are in bytes, unless a value is explicitly described as a number of bits.

## Header layout

| Offset | Size | Field       | Type      | Description |
|-------:|-----:|-------------|-----------|-------------|
| 0      | 4    | `magic`     | bytes     | ASCII `BLMF` (`42 4C 4D 46`) |
| 4      | 1    | `version`   | uint8     | Format version. This document describes version `1`. |
| 5      | 1    | `hash_id`   | uint8     | Hash algorithm, see [Hash algorithms](#hash-algorithms). |
| 6      | 2    | `reserved1` | bytes     | MUST be zero. |
| 8      | 8    | `k`         | uint64    | Number of bit positions set per element. MUST be ≥ 1. |
| 16     | 4    | `reserved2` | bytes     | MUST be zero. |
| 20     | 8    | `m`         | uint64    | Size of the bit array in **bits**. MUST be ≥ 1. |
| 28     | 8    | `seed1`     | uint64    | First hash seed. Meaning depends on `hash_id`. |
| 36     | 8    | `seed2`     | uint64    | Second hash seed. Meaning depends on `hash_id`. |

Total header size: **44 bytes**.

Reserved fields exist so that flags can be added later without moving other fields. A new meaning for a reserved field requires a new format version.

## Bit array

The bit array immediately follows the header and is exactly `(m + 7) / 8` bytes long (integer division, i.e. `m / 8` rounded up).

**Bit order.** Bit position `p` (where `0 ≤ p < m`) is stored in byte `p >> 3`, at bit `p & 7`, counting from the **least significant bit**:

```
byte index = p >> 3      (p / 8)
bit index  = p & 7       (p % 8)
mask       = 1 << (p & 7)
```

For example, position 27 is bit 3 of byte 3, mask `0b00001000`.

**Trailing bits.** When `m` is not a multiple of 8, the last byte contains `8 - (m % 8)` unused high bits. These bits MUST be zero.

## Checksum

The final 4 bytes are a CRC-32C checksum (Castagnoli polynomial) computed over **all preceding bytes**, header and bit array, stored as a little-endian uint32.

The checksum detects accidental corruption only. It does not protect against deliberate modification; see [Security considerations](#security-considerations).

## Element encoding

Elements are hashed as **the UTF-8 bytes of the string**, with no terminator, length prefix or normalization.

Strings that look identical but use different Unicode forms (for example a precomposed `é` versus `e` followed by a combining accent) produce different bytes and therefore different positions. Applications that need such strings to match MUST normalize them, for example to NFC, before adding or querying.

## Position derivation

For an element with UTF-8 bytes `data`, the `k` bit positions are derived from a single 128-bit hash split into two 64-bit values (the Kirsch–Mitzenmacher construction):

```
(h1, h2) = Hash(hash_id, seed1, seed2, data)

if h2 is even:
    h2 = h2 + 1

for i in 0 .. k-1:
    position[i] = (h1 + i * h2) mod 2^64 mod m
```

- All arithmetic is on unsigned 64-bit integers. The addition and multiplication wrap around modulo 2⁶⁴; this wrapping is intended and MUST be reproduced exactly.
- Forcing `h2` to be odd prevents `h2 = 0`, which would map all `k` positions to the same bit.
- **Add** sets the bit at every `position[i]`.
- **Contains** returns false as soon as any `position[i]` is 0, and true if all `k` are 1. A true result means "probably present"; a false result means "definitely absent".

Any change to this derivation, the bit order or the element encoding changes which bits an element maps to, and therefore requires a new format version.

## Hash algorithms

`hash_id` identifies an exact hash behavior: the algorithm variant, how the seeds are applied, and how the output is split into `h1` and `h2`. It does not identify a particular library. A different library that produces identical output uses the same ID.

| ID | Name              | Status   | Seed rules |
|---:|-------------------|----------|------------|
| 0  | —                 | Reserved | Never valid. A reader MUST reject it. |
| 1  | `murmur3-x64-128` | Active   | `seed1 == seed2`, and both MUST be < 2³². |
| 2  | `xxh3-128`        | Planned  | `seed1` is the seed; `seed2` MUST be 0. |

IDs are assigned explicitly and are **append-only**. An ID is never reused, even after its algorithm is retired.

### ID 1: murmur3-x64-128

The reference MurmurHash3_x64_128 algorithm with a single 32-bit seed, applied to both internal states. `h1` and `h2` are the first and second 64-bit halves of the output, as returned by the reference implementation.

With the seed rules above, the output matches common implementations such as Python's `mmh3.hash128` and Guava's `murmur3_128`, so filters can be built and queried from other languages.

The seed identifies the hash function so a filter can be rebuilt; it is **not** a secret key.

### ID 2: xxh3-128 (planned)

XXH3 128-bit, output as defined by xxHash v0.8.0 and later. The mapping of the two 64-bit output halves to `h1` and `h2` MUST be fixed in this document before the ID is used in any stored filter.

## Writer rules

A writer MUST:

1. Only serialize filters whose hasher has a registered `hash_id`. Filters using a non-persistable hash (for example a per-process randomized hash) cannot be serialized.
2. Write all reserved fields and unused trailing bits as zero.
3. Write a bit array of exactly `(m + 7) / 8` bytes.
4. Compute the CRC over the complete header and bit array.

Following these rules makes the encoding **canonical**: the same filter always serializes to the same bytes, so serialized filters can be compared or hashed directly.

## Reader rules

A reader MUST validate in this order and reject the input on the first failure:

1. The input is at least 48 bytes long (header plus checksum).
2. `magic` equals `BLMF`.
3. `version` is a version the reader supports.
4. `hash_id` is known to the reader.
5. `reserved1` and `reserved2` are zero.
6. `k ≥ 1` and `m ≥ 1`.
7. `m` does not exceed the reader's configured maximum (see [Limits](#limits)).
8. The seeds satisfy the rules for `hash_id`.
9. The total length equals `44 + (m + 7) / 8 + 4`.
10. The CRC-32C matches.
11. Unused trailing bits in the last byte are zero.

Only after all checks pass does the reader rebuild the hasher from `hash_id` and the seeds.

When reading from a stream, a reader SHOULD read the 44-byte header first, run checks 2–8, and only then allocate and read the remaining `(m + 7) / 8 + 4` bytes. This prevents a corrupted or malicious header from causing a huge allocation, and leaves the stream positioned exactly after the filter so that further data can follow.

A reader MUST NOT keep references into the caller's input buffer; the bit array is copied.

## Limits

Readers MUST enforce a maximum value for `m`. The limit is implementation-defined and SHOULD be documented by each implementation. It exists so that a corrupted or malicious header cannot trigger an unbounded memory allocation before the checksum can be verified.

## Compression

Compression is not part of this format. Applications MAY compress serialized filters with any general-purpose compressor. A reader can distinguish a gzip stream from an uncompressed filter by its first bytes (`1F 8B` for gzip, `42 4C 4D 46` for this format).

A filter near its design capacity has about half its bits set in a pseudorandom pattern and compresses poorly. Compression mainly helps for filters that are much emptier than they were sized for.

## Versioning

The format version MUST be incremented when any of the following change:

- the header layout or the meaning of a field,
- the bit order,
- the element encoding,
- the position derivation,
- the checksum algorithm or its coverage.

Adding a new `hash_id` does not require a new format version.

## Security considerations

- **Not confidential.** Anyone holding a serialized filter can test arbitrary candidate elements against it, because the hash algorithm and seeds are in the header. For low-entropy data such as e-mail addresses or phone numbers, most of the original set can be recovered by trying likely values. Do not share filters of sensitive data. Private set intersection requires a cryptographic protocol (for example ECDH-based blinding) in which the filter contains blinded values rather than raw elements.
- **Not tamper-proof.** CRC-32C detects accidental corruption, not deliberate changes. An attacker can modify a filter and recompute the checksum. Use an HMAC or signature over the serialized bytes when tampering is a concern.
- **Not flooding-resistant.** MurmurHash3 has known seed-independent multicollisions, so a secret seed does not stop an attacker from crafting inputs that fill a filter. Use a keyed hash such as SipHash when inputs are attacker-controlled.
- **Allocation limits.** See [Limits](#limits).

## Example

A filter with `m = 9586`, `k = 7`, hash ID 1 and seed 42 (`0x2A`):

- Bit array: `(9586 + 7) / 8 = 1199` bytes
- Total size: `44 + 1199 + 4 = 1247` bytes

Header bytes:

```
offset  bytes
0x00    42 4C 4D 46  01  01  00 00  07 00 00 00 00 00 00 00
0x10    00 00 00 00  72 25 00 00 00 00 00 00  2A 00 00 00
0x20    00 00 00 00  2A 00 00 00 00 00 00 00
```

| Bytes   | Field                 | Value |
|---------|-----------------------|-------|
| 0–3     | `magic`               | `BLMF` |
| 4       | `version`             | 1 |
| 5       | `hash_id`             | 1 (murmur3-x64-128) |
| 6–7     | `reserved1`           | 0 |
| 8–15    | `k`                   | 7 |
| 16–19   | `reserved2`           | 0 |
| 20–27   | `m`                   | 9586 (`0x2572`) |
| 28–35   | `seed1`               | 42 |
| 36–43   | `seed2`               | 42 |
