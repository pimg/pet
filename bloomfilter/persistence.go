package bloomfilter

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var ErrorNoPersistentHasher = errors.New("In order to use persistent the Hasher must implement PersistentHasher")

// TODO add godoc
// TODO add Seeds and ID to encoded string
func (bf *BloomFilter) Encode(rw io.ReadWriter) error {
	if !bf.persistentHasher() {
		return ErrorNoPersistentHasher
	}

	encoder := new(gob.Encoder)

	if bf.withCompression {
		gzipW := gzip.NewWriter(rw)
		defer func() {
			err := gzipW.Close()
			if err != nil {
				slog.Error("failed to close gzip writer", slog.Any("error", err))
			}
		}()
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

// TODO add godoc
// TODO add Seeds and ID to decoded struct
// TODO reconstruct the correct hasher with the correct seeds
func (bf *BloomFilter) Decode(rw io.ReadWriter) error {
	if !bf.persistentHasher() {
		return ErrorNoPersistentHasher
	}

	decoder := new(gob.Decoder)

	if bf.withCompression {
		gzipR, err := gzip.NewReader(rw)
		if err != nil {
			return fmt.Errorf("failed to create the gzip Reader: %v", err)
		}
		defer func() {
			err := gzipR.Close()
			if err != nil {
				slog.Error("failed to close gzip reader", slog.Any("error", err))
			}
		}()
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

func (bf *BloomFilter) persistentHasher() bool {
	switch bf.hasher.(type) {
	case PersistentHasher:
		return true
	default:
		return false
	}
}
