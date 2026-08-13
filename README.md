inspired by
https://maltsev.space/blog/008-bloom-filters-pt1

# TODO
- [v] implement Kirsch–Mitzenmacher optimization
- [v] implement persist and load from persist (serialize / deserialize)
- [v] implement compression in the persistence
- [v] implement thread safety

# Possible next steps
1. Private Set Intersection (PSI) Service
Build a Private Set Intersection tool that allows two parties to find common elements in their datasets (e.g., matching contact lists or ad audiences) without revealing the non-matching items to each other. 

The Bloom Filter Connection: Implement the classic Bloom Filter-based PSI protocol (or the more secure Garbled Bloom Filter variant).  You will encode one party’s set into a Bloom filter, encrypt it, and allow the other party to perform checks without learning the underlying data structure.
PET Concept: This is a foundational PET used in data clean rooms and secure collaboration.  It solves the "how do we collaborate without sharing raw data?" problem.

## Combine Bloomfilter with VRF and PSI
Yes, combining Bloom filters with Verifiable Random Functions (VRFs) creates a powerful primitive known as a Verifiable Bloom Filter (or sometimes a Cryptographically Secure Bloom Filter). This combination solves a critical trust issue: allowing a user to verify that a Bloom filter was constructed correctly and hasn't been manipulated to produce false negatives or targeted false positives, without revealing the underlying dataset. 

The Core Concept: Verifiable Membership
In a standard Bloom filter, you must trust the creator to have hashed the set elements correctly. If the creator is malicious, they could construct a filter that incorrectly reports "not present" for specific items (false negatives, which standard Bloom filters theoretically don't have but can be forced by a bad actor manipulating hash inputs) or manipulate the false positive rate. 

By integrating a VRF, the hash functions used to set bits in the filter become deterministic yet unpredictable and publicly verifiable:

Deterministic Indexing: Instead of standard hash functions ($H_1, H_2...$), the indices are generated using a VRF with a secret key $SK$ held by the data owner: $Index = VRF_{SK}(Element)$.
Public Verification: The owner publishes the VRF public key $PK$ and the resulting Bloom filter.
Proof of Correctness: When a user queries an element $x$, the owner provides the VRF output (the indices) and the VRF proof $\pi$.  The user can verify using $PK$ that the indices were indeed generated correctly from $x$ and the secret key, ensuring the filter wasn't tampered with to exclude $x$.

Privacy-Preserving Set Intersection (PSI)
In secure multi-party computation, two parties want to find common elements.

The VRF Role: A VRF can be used to map elements into a Bloom filter in a way that is oblivious yet verifiable. One party can prove they included their entire dataset in the filter without revealing the dataset itself.  This prevents a malicious party from omitting specific items from the intersection check to skew results.
Enhanced Security: It mitigates "partitioning attacks" where an adversary tries to learn about the set by querying specific elements and analyzing the filter's response patterns.
