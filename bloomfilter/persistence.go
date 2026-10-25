package bloomfilter

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

var ErrorNoPersistentHasher = errors.New("in order to use persistent the Hasher must implement PersistentHasher")
var ErrorNoSupportedHasher = errors.New("the hashing algorithm provided is not supported")
var ErrorChecksum = errors.New("checksum validation failed, binary data is corrupt")
var ErrorTruncated = errors.New("header is too short, binary data is corrupt")
var ErrorInvalidBloomFilter = errors.New("could not create header from BloomFilter")

var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// headerSize is the size of the binary header in bytes
// crcSize is the size of the checksum in bytes
const headerSize, crcSize = 44, 4

// WriteTo implements the WriterTo interface
// it marshals the bloomfilter and writes it to the provided Writer
// it returns the number of bytes written or an error
func (bf *BloomFilter) WriteTo(w io.Writer) (int64, error) {
	data, err := bf.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}

// MarshalBinary marshals the bloomfilter into a []bytes
// if follows the binary format specified in bloomfilter-format.md
func (bf *BloomFilter) MarshalBinary() ([]byte, error) {
	h, err := NewHeader(bf)
	if err != nil {
		return nil, ErrorInvalidBloomFilter // TODO consider returning abstracted error
	}

	buf := make([]byte, 0, binary.Size(h)+len(bf.filter)+4)
	buf, err = binary.Append(buf, binary.LittleEndian, h)
	if err != nil {
		return nil, err
	}
	buf = append(buf, bf.filter...)
	buf = binary.LittleEndian.AppendUint32(buf, crc32.Checksum(buf, castagnoli))
	return buf, nil
}

// ReadFrom implements the ReaderFrom interface
// it unmarshals the bloomfilter and reads from the provided Reader
// it returns the number of bytes read or an error
func (bf *BloomFilter) ReadFrom(r io.Reader) (int64, error) {
	buf := make([]byte, headerSize)

	n, err := io.ReadFull(r, buf)
	if err != nil {
		return int64(n), err
	}

	var h header
	if _, err := binary.Decode(buf, binary.LittleEndian, &h); err != nil {
		return int64(n), fmt.Errorf("decode header: %w", err)
	}

	err = h.Validate()
	if err != nil {
		return int64(n), err
	}

	rest := (h.M+7)/8 + crcSize
	buf = append(buf, make([]byte, rest)...)
	m, err := io.ReadFull(r, buf[headerSize:])
	if err != nil {
		return int64(m), err
	}

	total := n + m
	return int64(total), bf.UnmarshalBinary(buf)
}

// UnmarshalBinary unmarshals the bloomfilter into the Bloomfilter struct used as method receiver
// if follows the binary format specified in bloomfilter-format.md
// it returns an error
func (bf *BloomFilter) UnmarshalBinary(data []byte) error {
	if len(data) < headerSize+crcSize {
		return ErrorTruncated
	}

	body, sum := data[:len(data)-crcSize], data[len(data)-crcSize:]
	if crc32.Checksum(body, castagnoli) != binary.LittleEndian.Uint32(sum) {
		return ErrorChecksum
	}

	var h header
	if _, err := binary.Decode(data[:headerSize], binary.LittleEndian, &h); err != nil {
		return fmt.Errorf("decode header: %w", err)
	}

	loaded, err := BloomFilterFromHeader(&h, bytes.Clone(body[headerSize:]))
	if err != nil {
		return err
	}
	*bf = *loaded
	return nil
}
