package bloomfilter

import (
	"fmt"
	"hash/maphash"
)

const filterSize = 1024
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
		bf.filter[i] = 1
	}
}

func (bf *BloomFilter) Contains(value string) bool {
	for _, s := range bf.seeds {
		i := bf.index(s, value)
		if bf.filter[i] == 0 {
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
