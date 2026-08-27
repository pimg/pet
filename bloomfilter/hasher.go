package bloomfilter

import (
	"hash/maphash"
	"math/rand/v2"

	"github.com/twmb/murmur3"
)

type HashID uint8

const (
	HashUnknown    HashID = 0 // reserved: never valid in a file
	HashMurmur3x64 HashID = 1 // MurmurHash3_x64_128, seed1 == seed2, uint32 seed
)

// Hasher interface Sum128 takes []byte and returns two hashes in uint64
type Hasher interface {
	Sum128(data []byte) (h1, h2 uint64)
}

// PersistentHasher embeds Hasher and includes ID() to return the ID of the hashing algorithm and Seed() that returns the two seeds used for the two hashes
type PersistentHasher interface {
	Hasher
	ID() HashID
	Seed() (uint64, uint64)
}

// stdlib maphash hasher implementation
type MapHashHasher struct {
	seeds [2]maphash.Seed
}

func NewMapHashHasher() *MapHashHasher {
	return &MapHashHasher{
		seeds: [2]maphash.Seed{
			maphash.MakeSeed(),
			maphash.MakeSeed(),
		},
	}
}

func (h *MapHashHasher) Sum128(data []byte) (h1, h2 uint64) {
	h1 = maphash.Bytes(h.seeds[0], data)
	h2 = maphash.Bytes(h.seeds[1], data)

	// guard against multiplying by zero
	// and make sure h2 is odd number to increase likelyhood h2 is coprime with m
	if (h2 == 0) || (h2%2 == 0) {
		h2 += 1
	}

	return
}

// murmurhash3 hasher implementation

type Murmur3Hasher struct {
	seeds [2]uint64
}

func NewMurmur3Hasher() *Murmur3Hasher {
	return &Murmur3Hasher{
		seeds: [2]uint64{
			rand.Uint64(),
			rand.Uint64(),
		},
	}
}

func NewMurmur3HasherWithSeeds(s1, s2 uint64) *Murmur3Hasher {
	return &Murmur3Hasher{
		seeds: [2]uint64{
			s1,
			s2,
		},
	}
}

func (h *Murmur3Hasher) Sum128(data []byte) (h1, h2 uint64) {
	return murmur3.SeedSum128(h.seeds[0], h.seeds[1], data)
}

func (h *Murmur3Hasher) ID() HashID {
	return HashMurmur3x64
}

func (h *Murmur3Hasher) Seed() (uint64, uint64) {
	return h.seeds[0], h.seeds[1]
}
