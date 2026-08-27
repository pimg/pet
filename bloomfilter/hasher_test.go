package bloomfilter

import (
	"math/rand/v2"
	"testing"
)

func TestMapHashHasher(t *testing.T) {
	mh := NewMapHashHasher()

	h1, h2 := mh.Sum128([]byte("test"))
	ha, hb := mh.Sum128([]byte("test"))

	if h1 != ha || h2 != hb {
		t.Fatal("hashing the same value with the same seeds should produce the same hash")
	}
}

func TestMurmur3Hasher(t *testing.T) {
	mh := NewMurmur3Hasher()

	h1, h2 := mh.Sum128([]byte("test"))
	ha, hb := mh.Sum128([]byte("test"))

	if h1 != ha || h2 != hb {
		t.Fatal("hashing the same value with the same seeds should produce the same hash")
	}
}

func TestMurmur3HasherWithSeeds(t *testing.T) {
	mh := NewMurmur3HasherWithSeeds(rand.Uint64(), rand.Uint64())

	h1, h2 := mh.Sum128([]byte("test"))
	ha, hb := mh.Sum128([]byte("test"))

	if h1 != ha || h2 != hb {
		t.Fatal("hashing the same value with the same seeds should produce the same hash")
	}
}
