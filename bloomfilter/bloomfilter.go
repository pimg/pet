// package bloomfilter is a library implementing a bloom filter https://en.wikipedia.org/wiki/Bloom_filter
package bloomfilter

import (
	"errors"
	"fmt"
	"hash/maphash"
	"math"
	"math/bits"
	"sync"
)

// BloomFilter is the data structure holding the storage of the bloomfilter
// it contains parameters used for boundaries of the bloomfilter and a slice of seeds used for hashing
// An empty Bloom filter is a bit array of m bits, all set to 0.
// It is equipped with k different hash functions, which map set elements to one of the m possible array positions.
type BloomFilter struct {
	mSize           uint64
	kSize           uint64
	filter          []byte
	seeds           [2]maphash.Seed
	withCompression bool
	rwMu            *sync.RWMutex
}

type Parameters struct {
	K uint64
	M uint64
}

// Small sets m and k parameters for a small Bloomfilter that are alighed for optimal distribution
func Small() Parameters {
	return Parameters{
		7,
		9586,
	}
}

// Medium sets m and k parameters for a medium sized Bloomfilter that are alighed for optimal distribution
func Medium() Parameters {
	return Parameters{
		7,
		958506,
	}
}

// Large sets m and k parameters for a large sized Bloomfilter that are alighed for optimal distribution
func Large() Parameters {
	return Parameters{
		10,
		14377588,
	}
}

// WithSize option for setting custom m and k values
func WithSize(p Parameters) func(*BloomFilter) {
	return func(bf *BloomFilter) {
		bf.mSize = p.M
		bf.kSize = p.K
	}
}

// New initializes the bloomfilter with options for configuring the bloomfilter
func New(options ...func(*BloomFilter)) (*BloomFilter, error) {
	bf := &BloomFilter{}

	for _, option := range options {
		option(bf)
	}

	if bf.mSize == 0 {
		return nil, fmt.Errorf("bloomfilter initialized without filtersize (m)")
	}

	if bf.kSize == 0 {
		return nil, fmt.Errorf("bloomfilter initialized without kSize (k)")
	}

	bf.filter = make([]byte, (bf.mSize+7)/8)
	bf.seeds = [2]maphash.Seed{
		maphash.MakeSeed(),
		maphash.MakeSeed(),
	}

	bf.rwMu = new(sync.RWMutex)

	return bf, nil
}

// NewFromEstimate creates a new BloomFilter but calculates the filterSize (m) and kSize (k)
// from the expectedItems stored in the bloomfilter as well as the false positive rate as float (0.03) is 3% false positive rate
func NewFromEstimate(expectedItems int, falsePositiveRate float64) (*BloomFilter, error) {
	if expectedItems <= 0 {
		return nil, errors.New("expected items must be > 0")
	}

	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return nil, errors.New("falsePositiveRate must be > 0 and < 1")
	}

	m := -float64(expectedItems) * math.Log(falsePositiveRate) / (math.Ln2 * math.Ln2)
	k := (m / float64(expectedItems)) * math.Ln2
	if k < 1 {
		k = 1
	}

	filterSize := uint64(math.Ceil(m))
	kSize := uint64(math.Round(k))

	return New(WithSize(Parameters{M: filterSize, K: kSize}))
}

// Set inserts a string value in the bloomfilter
func (bf *BloomFilter) Add(value string) {
	bf.rwMu.Lock()
	defer bf.rwMu.Unlock()

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
	bf.rwMu.RLock()
	defer bf.rwMu.RUnlock()

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
	// and make sure h2 is odd number to increase likelyhood h2 is coprime with m
	if (h2 == 0) || (h2%2 == 0) {
		h2 += 1
	}

	for i := range bf.kSize {
		index := (h1 + i*h2) % bf.mSize
		if !op(index) {
			return false
		}
	}

	return true
}

// FillRatio returns the percentage of bits set in the bloomfilter
func (bf *BloomFilter) FillRatio() float64 {
	total := 0
	for _, b := range bf.filter {
		total += bits.OnesCount8(b)
	}

	return float64(total) / float64(bf.mSize)
}

// CurrentFalsePositiveRate calculates the curent false positive rate based on the current Fill Ratio
func (bf *BloomFilter) CurrentFalsePositiveRate() float64 {
	return math.Pow(bf.FillRatio(), float64(bf.kSize))
}
