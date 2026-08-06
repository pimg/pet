// package bloomfilter is a library implementing a bloom filter https://en.wikipedia.org/wiki/Bloom_filter
package bloomfilter

import (
	"fmt"
	"hash/maphash"
	"math"
)

// BloomFilter is the data structure holding the storage of the bloomfilter
// it contains parameters used for boundaries of the bloomfilter and a slice of seeds used for hashing
// An empty Bloom filter is a bit array of m bits, all set to 0.
// It is equipped with k different hash functions, which map set elements to one of the m possible array positions.
type BloomFilter struct {
	filterSize uint64
	kSize      uint64
	filter     []byte
	seeds      []maphash.Seed
}

type parameters struct {
	k uint64
	m uint64
}

// New creates a bloomfilter with one of the default parameter groups
func New(conf parameters) *BloomFilter {
	return new(conf.m, conf.k)
}

// Small sets m and k parameters for a small Bloomfilter that are alighed for optimal distribution
func Small() parameters {
	return parameters{
		7,
		9586,
	}
}

// Medium sets m and k parameters for a medium sized Bloomfilter that are alighed for optimal distribution
func Medium() parameters {
	return parameters{
		7,
		958506,
	}
}

// Large sets m and k parameters for a large sized Bloomfilter that are alighed for optimal distribution
func Large() parameters {
	return parameters{
		10,
		14377588,
	}
}

// new creates a bloomfilter with the filterSize (m) and kSize (k)
func new(filterSize, kSize uint64) *BloomFilter {
	bf := &BloomFilter{
		filterSize: filterSize,
		kSize:      kSize,
		filter:     make([]byte, filterSize),
		seeds:      make([]maphash.Seed, 2),
	}
	for i := range bf.seeds {
		bf.seeds[i] = maphash.MakeSeed()
	}

	return bf
}

// NewFromEstimate creates a new BloomFilter but calculates the filterSize (m) and kSize (k)
// from the expectedItems stored in the bloomfilter as well as the false positive rate as float (0.03) is 3% false positive rate
func NewFromEstimate(expectedItems int, falsePositiveRate float64) *BloomFilter {
	m := -float64(expectedItems) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2)
	k := (m / float64(expectedItems)) * math.Ln2
	if k < 1 {
		k = 1
	}

	filterSize := uint64(math.Round(m))
	kSize := uint64(math.Round(k))

	return new(filterSize, kSize)
}

// Set inserts a string value in the bloomfilter
func (bf *BloomFilter) Set(value string) {
	setOp := func(index uint64) bool {
		// i >> 3 (i / 8) get the actual byte that is holding the index
		// i & 7 determines which bit in the byte (a byte stores 0-7 bits) the result is the position in the byte between 0 and 7
		// 1 << (i & 7) builds a bitmask with all zero's except for the bit at position (i & 7)
		// |= sets the bit set at one determined in the bitmask in the byte and ignoring the values in the other bit positions of the byte
		bf.filter[index>>3] |= 1 << (index & 7)
		return true
	}
	bf.index(value, setOp)
}

// Contains checks if a string is stored in the bloomfilter
// It returns true if the item is probably in the bloomfilter.
// It returns false if the string is definitely not stored in the bloomfilter
func (bf *BloomFilter) Contains(value string) bool {
	containsOp := func(index uint64) bool {
		// i >> 3 (i / 8) get the actual byte that is holding the index
		// i & 7 determines which bit in the byte (a byte stores 0-7 bits) the result is the position in the byte between 0 and 7
		// 1 << (i & 7) builds a bitmask with all zero's except for the bit at position (i & 7)
		// & tests if the bit in the position of the bitmask is set to 1 if it is 0 it means the bit was clear and the bloomfilter doesn not contains the value hence the return false
		if bf.filter[index>>3]&(1<<(index&7)) == 0 {
			return false
		}
		return true
	}
	return bf.index(value, containsOp)
}

func (bf *BloomFilter) index(value string, op func(index uint64) bool) bool {
	h1 := maphash.String(bf.seeds[0], value)
	h2 := maphash.String(bf.seeds[1], value)

	// guard against multiplying by zero
	// and make sure h2 is negative number to increase likelyhood h2 is coprime with m
	if (h2 == 0) || (h2%2 == 0) {
		h2 += 1
	}

	// guard against multiplying by zero
	if h1 == 0 {
		h1 = 1
	}

	for i := range bf.kSize {
		index := (h1 + i*h2) % bf.filterSize
		if !op(index) {
			return false
		}
	}

	return true
}

// Print prints the entire content of the bloomfilter to stdout
// it can be used for manual verification and debugging
// the print prints the bit layout grouped by bytes with 4 bytes on a line
func (bf *BloomFilter) Print() {
	fmt.Println("Bloom filter content:")
	for i, b := range bf.filter {
		fmt.Printf("%08b ", b)
		if (i+1)%4 == 0 {
			fmt.Println()
		}
	}
}
