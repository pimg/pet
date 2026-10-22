package bloomfilter

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"
)

var ErrorNoPersistentHasher = errors.New("In order to use persistent the Hasher must implement PersistentHasher")
var ErrorNoSupportedHasher = errors.New("The hashing algorithm provided is not supported")

// TODO add godoc
func (bf *BloomFilter) Encode(rw io.ReadWriter) error {
	if !bf.persistentHasher() {
		return ErrorNoPersistentHasher
	}

	encoder := gob.NewEncoder(rw)

	err := encoder.Encode(bf.filter)
	if err != nil {
		return fmt.Errorf("failed to encode Bloomfilter: %v", err)
	}

	return nil
}

// TODO add godoc
// TODO reconstruct the correct hasher with the correct seeds
func (bf *BloomFilter) Decode(rw io.ReadWriter) error {
	if !bf.persistentHasher() {
		return ErrorNoPersistentHasher
	}

	decoder := gob.NewDecoder(rw)

	err := decoder.Decode(&bf.filter)
	if err != nil {
		return fmt.Errorf("failed to decode Bloomfilter: %v", err)
	}

	return nil
}

func (bf *BloomFilter) persistentHasher() bool {
	switch bf.hasher.(type) {
	case PersistentHasher:
		return true
	default:
		return false
	}
}
