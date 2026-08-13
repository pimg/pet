package bloomfilter

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"log/slog"
)

func (bf *BloomFilter) Encode(rw io.ReadWriter) error {
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

func (bf *BloomFilter) Decode(rw io.ReadWriter) error {
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
