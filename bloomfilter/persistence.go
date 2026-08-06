package bloomfilter

import (
	"encoding/gob"
	"fmt"
	"io"
)

// TODO consider io.ReadWriterCloser
func (bf *BloomFilter) Encode(rw io.ReadWriter) error {
	encoder := gob.NewEncoder(rw)
	err := encoder.Encode(bf.filter)
	if err != nil {
		return fmt.Errorf("failed to encode Bloomfilter: %v", err)
	}

	return nil
}

// TODO consider io.ReadWriteCloser
func (bf *BloomFilter) Decode(rw io.ReadWriter) error {
	decoder := gob.NewDecoder(rw)
	err := decoder.Decode(&bf.filter)
	if err != nil {
		return fmt.Errorf("failed to decode Bloomfilter: %v", err)
	}

	return nil
}

// TODO create Options for New
