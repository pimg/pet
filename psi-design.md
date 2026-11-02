# Private Set Intersection: Phased Design

This document describes how the Bloom filter library is extended into a private set intersection (PSI) protocol, and eventually a post-quantum one. It is a learning project. The design favors small, testable steps over performance or production hardening.

The approach mirrors how the Bloom filter itself was built: start with the simplest version that works end to end, put a stable interface around each part that will change, and replace one part at a time with a test harness that confirms nothing broke.

## Goal and roles

Two parties each hold a set of byte strings:

- The **server** holds set `X`.
- The **client** holds set `Y`.

At the end, the client learns `X ∩ Y` and nothing else about `X`. The server learns nothing about `Y`. Both learn the sizes `|X|` and `|Y|`, which is standard for PSI and needed to size the filter.

In every phase, the server's final message is a Bloom filter in the existing exchange format, and the client tests its own values against it. What changes from phase to phase is **what goes into the filter**, and therefore what an attacker can learn from it.

## Security model

- **Semi-honest parties.** Both parties follow the protocol but try to learn more from what they see. Protection against parties who deviate from the protocol (malicious security) is out of scope.
- **Network attackers** are handled by the transport, not the PSI protocol: everything runs over TLS (Phase 2 onward).
- **Security parameters:**
  - Computational: κ = 128 bits. Symmetric primitives use SHA3-256 / SHAKE256, which keeps a comfortable margin against quantum (Grover) attacks.
  - Statistical: λ = 40 bits. The probability that the client's result contains any wrong element is at most 2⁻⁴⁰.

## Phase overview

| Phase | What goes in the filter | Protects against | Quantum-safe | New component |
|------:|-------------------------|------------------|:------------:|---------------|
| 0 | — | — | — | Byte-element support, sizing for λ |
| 1 | Raw items | Nothing (deliberately insecure) | — | Protocol interfaces, framing, test harness |
| 2 | Raw items | Network eavesdroppers | Transport only | TLS transport |
| 3 | DH-OPRF outputs `H(x)^s` | Curious client and server | No | Elliptic-curve OPRF |
| 4 | Chase–Miao OPRF outputs | Curious client and server | No (base OT) | OT-based OPRF with a test-only base OT |
| 5 | Chase–Miao OPRF outputs | Curious client and server | No (base OT) | Real classical base OT |
| 6 | Chase–Miao OPRF outputs | Curious client and server | **Yes** | KEM-based base OT |

Each phase is complete on its own: it produces correct intersections and has passing tests before the next one starts.

---

## Phase 0: Prepare the Bloom filter

The filter needs two small changes before it can carry PSI tokens.

**Byte elements.** OPRF outputs are arbitrary 32-byte values, not text. Add `AddBytes([]byte)` and `ContainsBytes([]byte)` alongside the string methods. A Go string can hold arbitrary bytes, but the format specification says elements are UTF-8 strings, so make it explicit:

- Update the specification: an element is a **byte sequence**; string elements are hashed as their UTF-8 bytes.
- This does not change which bits an element maps to, so the format version stays 1.

**Sizing for λ.** The client checks `|Y|` values, so the per-query false positive rate must be `p = 2⁻λ / |Y|` for the total error to stay below 2⁻λ. For `|Y| = 2²⁰` that is `p = 2⁻⁶⁰`:

- about 1.44 · 60 ≈ 87 bits per server element,
- `k ≈ 60` hash positions.

Add a helper `SizeForPSI(serverItems, clientItems int, lambda int)` that returns `m` and `k`, and a test that checks the numbers against the formulas.

**Done when:** byte-element round trips pass, and the sizing helper is tested.

---

## Phase 1: Naive PSI (insecure baseline)

The server puts its raw items into a filter and sends it. The client tests its own items. This leaks the server's set to a dictionary attack, which is the point: it gives a working end-to-end pipeline, and a test that demonstrates *why* the later phases exist.

### Interfaces

These stay stable through every later phase:

```go
type Server interface {
    Serve(ctx context.Context, conn io.ReadWriter, items [][]byte) error
}

type Client interface {
    Intersect(ctx context.Context, conn io.ReadWriter, items [][]byte) ([][]byte, error)
}
```

Each phase is a separate package (`psi/naive`, `psi/ecdh`, `psi/chasemiao`) implementing both. Shared code lives in `psi/wire`.

### Wire framing

Messages are framed as:

```
type (uint8) | length (uint32, little-endian) | payload
```

- Readers enforce a maximum length before allocating, as the filter format does.
- The Bloom filter message payload is the existing filter format, written with `WriteTo` and read with `ReadFrom`.

### Messages

1. **Hello** (client → server): protocol ID, protocol version, `|Y|`, λ.
2. **Hello** (server → client): protocol ID, protocol version, `|X|`.
3. **Filter** (server → client): Bloom filter containing every item of `X`, sized with `SizeForPSI`.

The client returns every `y` for which `ContainsBytes(y)` is true.

### Test harness

Built once, reused by every phase:

- **In-memory transport:** `net.Pipe()`, with server and client running in separate goroutines.
- **Correctness:** sets with a known overlap (for example 1000 items each, 100 shared). The result must equal the true intersection exactly.
- **Edge cases:** empty sets, identical sets, disjoint sets.
- **False-positive budget:** with a small λ (for example 8), verify the observed error rate matches the configured one. At λ = 40 errors are too rare to observe.
- **Dictionary attack (negative test):** a client that takes the received filter and tests 10,000 guessed items recovers the server items among them. This test *passes* in Phase 1, showing the leak. From Phase 3 on, the equivalent attack must fail.

**Done when:** all harness tests pass, including the attack demonstration.

---

## Phase 2: Transport security

Wrap the connection in TLS. Since Go 1.24, `crypto/tls` uses the hybrid `X25519MLKEM768` key exchange by default, so this phase gives protection against "harvest now, decrypt later" at almost no cost.

This is deliberately a separate phase to make the boundary clear: TLS protects against **outsiders**, but the client still receives the server's raw items. The dictionary-attack test still passes. Privacy from the other party is the job of the OPRF, which arrives in Phase 3.

The PSI interfaces don't change, because they take an `io.ReadWriter` and a `*tls.Conn` is one.

**Done when:** the harness runs over a TLS connection (with a test certificate), and a test confirms the negotiated group is `X25519MLKEM768`.

---

## Phase 3: DH-OPRF PSI (first real PSI)

The first protocol with real privacy, based on elliptic-curve Diffie-Hellman. It is **not** quantum-safe, but it is simple, well understood, and it serves as a **reference implementation**: later phases must produce the same intersections.

### Building blocks

- The ristretto255 group, via `github.com/gtank/ristretto255`.
- Hash-to-group: SHA-512 of the item, mapped with `FromUniformBytes`.
- Fresh random scalars `s` (server) and `c` (client) **for every session**.

### Flow

1. Client sends `H(y)^c` for each `y ∈ Y`, in random order.
2. Server returns `(H(y)^c)^s` for each, in the same order.
3. Client removes its blinding: `((H(y)^c)^s)^(1/c) = H(y)^s`.
4. Server sends a filter containing `H(x)^s` for each `x ∈ X`, encoded as 32 bytes.
5. Client tests each `H(y)^s` against the filter.

The client can't test guesses, because it can't compute `H(guess)^s` without the server's key. The server sees only blinded points.

**Done when:** the harness passes, and the dictionary-attack test now *fails* to recover server items.

---

## Phase 4: OT-based OPRF (Chase–Miao), test-only base OT

The DH-OPRF is replaced with the multi-point OPRF from Chase and Miao, "Private Set Intersection in the Internet Setting From Lightweight Oblivious PRF" (CRYPTO 2020). After a small number of base OTs, everything is symmetric cryptography, which is what later makes the protocol post-quantum.

To keep this phase about the OPRF and not the OT, the base OTs come from a **test-only dealer** (see [Base OT interface](#base-ot-interface)). This is insecure by design and must never be used outside tests.

### Parameters

- `m`: number of matrix rows, about `|Y|`.
- `w`: matrix width, which is also the number of base OTs. The paper tabulates `w` per set size; it is several hundred.
- `F`: a PRF with output in `[m]^w`, for example SHAKE256 keyed with `k`.
- `H1`, `H2`: hash functions, built from SHA3-256 with domain separation.

### Flow

1. **Base OTs.** The server picks a random choice vector `s ∈ {0,1}^w`. The parties run `w` base OTs in which the **client is the OT sender** with random seed pairs `(k_i⁰, k_i¹)`, and the server receives `k_i^(s_i)`.
2. **Client builds the matrix.** The client picks a PRF key `k` and sends it. For each `y`, it computes `v = F_k(H1(y))` and sets `D[v_i][i] = 0` in an all-ones `m × w` matrix `D`.
3. **Column transfer.** The client sets `A_i = PRG(k_i⁰)` and sends `u_i = PRG(k_i⁰) ⊕ PRG(k_i¹) ⊕ D_i` for each column `i`.
4. **Server derives its columns.** If `s_i = 0`, the server's column is `C_i = PRG(k_i⁰) = A_i`. If `s_i = 1`, it is `C_i = PRG(k_i¹) ⊕ u_i = A_i ⊕ D_i`.
5. **Server tokens.** For each `x`, the server computes `v = F_k(H1(x))` and the token `H2(C_1[v_1] ‖ … ‖ C_w[v_w])`, and inserts it into the filter.
6. **Client tokens.** For each `y`, the client computes `H2(A_1[v_1] ‖ … ‖ A_w[v_w])` and tests it against the filter.

For `y ∈ Y`, every `D[v_i][i]` is 0, so the server's columns agree with `A` at those positions and the tokens match. For other values, they differ wherever `s_i = 1` and `D` has a 1, so the server's token is pseudorandom to the client.

The filter replaces the paper's list of truncated hashes. It is smaller (about 1.44 bits per bit of security instead of a full hash per item), at the cost of about `k` probes per lookup.

**Done when:** the harness passes, and for the same input sets this phase returns the same intersection as Phase 3.

---

## Phase 5: Real classical base OT

Replace the test dealer with a real base OT: the "simplest OT" of Chou and Orlandi (2015), implemented on ristretto255, which is already a dependency from Phase 3.

Only the base OT implementation changes. The Chase–Miao code calls the same interface.

**Done when:** the harness passes with the new base OT, with results identical to Phase 4.

---

## Phase 6: Post-quantum base OT

Replace the classical base OT with a KEM-based one, following the "endemic OT" construction of Masny and Rindal (CCS 2019), which includes a Kyber-based instantiation.

Once this is in place, every component is post-quantum:

- base OTs from a lattice KEM,
- OT extension, OPRF, tokens and transport keys from symmetric primitives and hybrid TLS.

### Implementation note

These OT constructions need to manipulate public keys directly, for example adding random values to a public key element by element. Go's `crypto/mlkem` only exposes key generation, encapsulation and decapsulation on encoded keys, so it is not enough on its own. Expect to implement the required lattice arithmetic, or use a library that exposes it. This is the most demanding phase, which is why everything else is in place and tested before it.

**Done when:** the harness passes with the KEM-based base OT, with results identical to Phases 4 and 5.

---

## Base OT interface

The base OT is the one component replaced three times, so it gets its own interface, in the same spirit as the pluggable hasher:

```go
// Random OT: the sender gets n pairs of random seeds,
// the receiver gets one seed of each pair, selected by its choice bits.
type BaseOTSender interface {
    Send(ctx context.Context, conn io.ReadWriter, n int) ([][2][32]byte, error)
}

type BaseOTReceiver interface {
    Receive(ctx context.Context, conn io.ReadWriter, choices []bool) ([][32]byte, error)
}
```

| Implementation  | Phase | Security |
|-----------------|------:|----------|
| `ot/dealer`     | 4     | None. Both sides derive their outputs from a shared test seed. Tests only. |
| `ot/simplest`   | 5     | Classical (Chou–Orlandi, ristretto255). |
| `ot/kem`        | 6     | Post-quantum (Masny–Rindal, lattice KEM). |

Each implementation gets its own unit test that checks the OT property directly: for every `i`, the receiver's seed equals `pairs[i][choices[i]]`. Bugs then show up in the OT test, not as a mysterious wrong intersection.

## Cross-cutting rules

- **Fresh randomness per session.** OPRF keys, scalars, choice vectors and seeds are never reused across sessions. Reuse lets a client combine evaluations from several runs.
- **Domain separation.** Every hash use gets a distinct prefix, for example `"psi/v1/H1"` and `"psi/v1/H2"`, so outputs of one role can never be mistaken for another.
- **Versioned protocol IDs.** The Hello message carries a protocol ID per phase and a version, using the same append-only rules as the filter's hash IDs.
- **Fail closed.** Any malformed, oversized or unexpected message ends the session with an error. No partial results.

## Out of scope

- Malicious security (parties that deviate from the protocol).
- Reusing one server filter for many clients (unbalanced or asymmetric PSI).
- PSI cardinality (learning only `|X ∩ Y|`).
- More than two parties.

Each of these is a possible follow-up once Phase 6 works.

## References

- Chase, Miao. *Private Set Intersection in the Internet Setting From Lightweight Oblivious PRF.* CRYPTO 2020.
- Chou, Orlandi. *The Simplest Protocol for Oblivious Transfer.* LATINCRYPT 2015.
- Masny, Rindal. *Endemic Oblivious Transfer.* ACM CCS 2019.
- Kolesnikov, Kumaresan, Rosulek, Trieu. *Efficient Batched Oblivious PRF with Applications to Private Set Intersection.* ACM CCS 2016 (the per-bin OPRF this design does not use).
- Bloom filter exchange format: `bloomfilter-format.md`.
