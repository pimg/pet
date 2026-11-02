// package bloomfilter is a library implementing a bloom filter https://en.wikipedia.org/wiki/Bloom_filter
package bloomfilter

import (
	"errors"
	"math"
	"math/bits"
	"sync"
)

// BloomFilter is the data structure holding the storage of the bloomfilter
// it contains parameters used for boundaries of the bloomfilter and a slice of seeds used for hashing
// An empty Bloom filter is a bit array of m bits, all set to 0.
// It is equipped with k different hash functions, which map set elements to one of the m possible array positions.
type BloomFilter struct {
	mSize  uint64
	kSize  uint64
	filter []byte
	hasher Hasher
	rwMu   *sync.RWMutex
}

type Parameters struct {
	K uint64
	M uint64
}

type BloomFilterOption func(bf *BloomFilter) error

// parameters used for PSI
// DefaultLambda is the Default lambda that can be used for creating a new BloomFilter used for PSI
const DefaultLambda = 40
const maxFilterBits = 1 << 33

var ErrNoMSize = errors.New("bloomfilter initialized without filtersize (m)")
var ErrNoKSize = errors.New("bloomfilter initialized without kSize (k)")
var ErrInvalidExpectedItems = errors.New("expected items must be > 0")
var ErrInvalidFalsePositiveRate = errors.New("falsePositiveRate must be > 0 and < 1")
var ErrFilterTooLarge = errors.New("filter will be too large with the current serverItems")
var ErrInvalidLambda = errors.New("lambda must be between 1 and 128")
var ErrInvalidPSIParameters = errors.New("cannot create a bloomfilter with PSI parameters that are <= 0")

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
func WithSize(p Parameters) BloomFilterOption {
	return func(bf *BloomFilter) error {
		bf.mSize = p.M
		bf.kSize = p.K
		return nil
	}
}

// WithHasher is an option for configuring the Bloomfilter with a caller provided Hasher implementation
func WithHasher(h Hasher) BloomFilterOption {
	return func(bf *BloomFilter) error {
		bf.hasher = h
		return nil
	}
}

func ForPSI(serverItems, clientItems int, lambda int) BloomFilterOption {
	return func(bf *BloomFilter) error {
		m, k, err := psiParameters(serverItems, clientItems, lambda)
		if err != nil {
			return err
		}

		bf.mSize = m
		bf.kSize = k

		return nil
	}
}

func psiParameters(serverItems, clientItems int, lambda int) (uint64, uint64, error) {
	if serverItems <= 0 || clientItems <= 0 || lambda <= 0 {
		return 0, 0, ErrInvalidPSIParameters
	}

	if lambda < 1 || lambda > 128 {
		return 0, 0, ErrInvalidLambda
	}

	e := float64(lambda) + math.Log2(float64(clientItems))
	m := math.Ceil(float64(serverItems) * e / math.Ln2)
	if m > maxFilterBits {
		return 0, 0, ErrFilterTooLarge
	}

	k := math.Ceil(e)

	return uint64(m), uint64(k), nil
}

// New initializes the bloomfilter with options for configuring the bloomfilter
func New(options ...BloomFilterOption) (*BloomFilter, error) {
	bf := &BloomFilter{}

	for _, option := range options {
		err := option(bf)
		if err != nil {
			return nil, err
		}
	}

	if bf.mSize == 0 {
		return nil, ErrNoMSize
	}

	if bf.kSize == 0 {
		return nil, ErrNoKSize
	}

	bf.filter = make([]byte, (bf.mSize+7)/8)

	// MapHashHasher is the default hasher
	if bf.hasher == nil {
		bf.hasher = NewMapHashHasher()
	}

	bf.rwMu = new(sync.RWMutex)

	return bf, nil
}

// NewFromEstimate creates a new BloomFilter but calculates the filterSize (m) and kSize (k)
// from the expectedItems stored in the bloomfilter as well as the false positive rate as float (0.03) is 3% false positive rate
func NewFromEstimate(expectedItems int, falsePositiveRate float64) (*BloomFilter, error) {
	if expectedItems <= 0 {
		return nil, ErrInvalidExpectedItems
	}

	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return nil, ErrInvalidFalsePositiveRate
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

	bf.index([]byte(value), bf.addFunc())
}

// Set inserts a [byte value in the bloomfilter
func (bf *BloomFilter) AddBytes(value []byte) {
	bf.rwMu.Lock()
	defer bf.rwMu.Unlock()

	bf.index(value, bf.addFunc())
}

func (bf *BloomFilter) addFunc() func(index uint64) bool {
	return func(index uint64) bool {
		// i >> 3 (i / 8) get the actual byte that is holding the index
		// i & 7 determines which bit in the byte (a byte stores 0-7 bits) the result is the position in the byte between 0 and 7
		// 1 << (i & 7) builds a bitmask with all zero's except for the bit at position (i & 7)
		// |= sets the bit set at one determined in the bitmask in the byte and ignoring the values in the other bit positions of the byte
		bf.filter[index>>3] |= 1 << (index & 7)
		return true
	}
}

// Contains checks if a string is stored in the bloomfilter
// It returns true if the item is probably in the bloomfilter.
// It returns false if the string is definitely not stored in the bloomfilter
func (bf *BloomFilter) Contains(value string) bool {
	bf.rwMu.RLock()
	defer bf.rwMu.RUnlock()

	return bf.index([]byte(value), bf.containsFunc())
}

// Contains checks if a []byte is stored in the bloomfilter
// It returns true if the item is probably in the bloomfilter.
// It returns false if the []byte is definitely not stored in the bloomfilter
func (bf *BloomFilter) ContainsBytes(value []byte) bool {
	bf.rwMu.RLock()
	defer bf.rwMu.RUnlock()

	return bf.index(value, bf.containsFunc())
}

func (bf *BloomFilter) containsFunc() func(index uint64) bool {
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
	return containsOp
}

func (bf *BloomFilter) index(value []byte, op func(index uint64) bool) bool {
	h1, h2 := bf.hasher.Sum128(value)
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
