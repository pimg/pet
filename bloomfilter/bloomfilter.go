package bloomfilter

import (
	"fmt"
	"hash/maphash"
)

const filterSize = 1024 / 8 // temp divide by 8 to have consistent test results when implementing bit packing
const kSize = 7

type BloomFilter struct {
	filter [filterSize]byte
	seeds  [kSize]maphash.Seed
}

func New() *BloomFilter {
	bf := &BloomFilter{}
	for i := range bf.seeds {
		bf.seeds[i] = maphash.MakeSeed()
	}

	return bf
}

func (bf *BloomFilter) Set(value string) {
	for _, s := range bf.seeds {
		i := bf.index(s, value)
		// i >> 3 (i / 8) get the actual byte that is holding the index
		// i & 7 determines which bit in the byte (a byte stores 0-7 bits) the result is the position in the byte between 0 and 7
		// 1 << (i & 7) builds a bitmask with all zero's except for the bit at position (i & 7)
		// |= sets the bit set at one determined in the bitmask in the byte and ignoring the values in the other bit positions of the byte
		bf.filter[i>>3] |= 1 << (i & 7)
	}
}

func (bf *BloomFilter) Contains(value string) bool {
	for _, s := range bf.seeds {
		i := bf.index(s, value)
		// i >> 3 (i / 8) get the actual byte that is holding the index
		// i & 7 determines which bit in the byte (a byte stores 0-7 bits) the result is the position in the byte between 0 and 7
		// 1 << (i & 7) builds a bitmask with all zero's except for the bit at position (i & 7)
		// & tests if the bit in the position of the bitmask is set to 1 if it is 0 it means the bit was clear and the bloomfilter doesn not contains the value hence the return false
		if bf.filter[i>>3]&(1<<(i&7)) == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) index(s maphash.Seed, value string) uint64 {
	return maphash.String(s, value) % filterSize
}

func (bf *BloomFilter) Print() {
	fmt.Printf("bloom filter content: %d\n", bf.filter)
}
