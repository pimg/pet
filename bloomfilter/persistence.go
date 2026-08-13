package bloomfilter

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
)

func (bf *BloomFilter) Encode(rw io.ReadWriter) error {
	encoder := new(gob.Encoder)

	if bf.withCompression {
		gzipW := gzip.NewWriter(rw)
		defer gzipW.Close()
		encoder = gob.NewEncoder(gzipW)
	} else {
		encoder = gob.NewEncoder(rw)
	}

	err := encoder.Encode(bf.filter)
	if err != nil {
		return fmt.Errorf("failed to encode Bloomfilter: %v", err)
	}

	return nil
}

func (bf *BloomFilter) Decode(rw io.ReadWriter) error {
	decoder := new(gob.Decoder)

	if bf.withCompression {
		gzipR, err := gzip.NewReader(rw)
		if err != nil {
			return fmt.Errorf("failed to create the gzip Reader: %v", err)
		}
		defer gzipR.Close()
		decoder = gob.NewDecoder(gzipR)
	} else {
		decoder = gob.NewDecoder(rw)
	}

	err := decoder.Decode(&bf.filter)
	if err != nil {
		return fmt.Errorf("failed to decode Bloomfilter: %v", err)
	}

	return nil
}

func WithCompression() func(*BloomFilter) {
	return func(bf *BloomFilter) {
		bf.withCompression = true
	}
}
