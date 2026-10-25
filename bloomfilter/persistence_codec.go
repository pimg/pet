package bloomfilter

import (
	"errors"
	"sync"
)

// header is the codec structure used when encoding/decoding the bloomfilter
type header struct {
	Magic     [4]byte
	Version   uint8
	HashID    uint8
	Reserved1 [2]byte
	K         uint64
	Reserved2 [4]byte
	M         uint64
	Seed1     uint64
	Seed2     uint64
}

const (
	Version1 = 1
)

var ErrorInvalidHeaderFieldLength = errors.New("header field has an invalid length")
var ErrorInvalidHeaderMagic = errors.New("invalid magic header: should be BLMF")
var ErrorInvalidHeaderVersion = errors.New("invalid header version")
var ErrorInvalidHeaderHashID = errors.New("invalid header HashID")
var ErrorInvalidHeaderReserved = errors.New("invalid header Reserved should contain two 0x00 bytes")
var ErrorInvalidHeaderKM = errors.New("invalid header parameter k and m must be larger than 1")

// Validate validates the header
func (h *header) Validate() error {
	return errors.Join(
		validateMagic(h.Magic),
		validateVersion(h.Version),
		validateHashID(h.HashID),
		validateReserved1(h.Reserved1),
		validateKM(h.K),
		validateReserved2(h.Reserved2),
		validateKM(h.M),
	)
}

func validateMagic(m [4]byte) error {
	if m != [4]byte{'B', 'L', 'M', 'F'} {
		return ErrorInvalidHeaderMagic
	}

	return nil
}

func validateVersion(v uint8) error {
	switch v {
	case Version1:
		return nil
	default:
		return ErrorInvalidHeaderVersion
	}
}

func validateHashID(hid uint8) error {
	switch hid {
	case uint8(HashMurmur3x64):
		return nil
	default:
		return ErrorInvalidHeaderHashID
	}
}

func validateReserved1(r [2]byte) error {
	if r != [2]byte{} {
		return ErrorInvalidHeaderReserved
	}
	return nil
}

func validateReserved2(r [4]byte) error {
	if r != [4]byte{} {
		return ErrorInvalidHeaderReserved
	}
	return nil
}

func validateKM(km uint64) error {
	if km <= 1 {
		return ErrorInvalidHeaderKM
	}
	return nil
}

// NewHeader creates the header from the BloomFilter
// it returs a pointer to the header or an error
func NewHeader(bf *BloomFilter) (*header, error) {
	m := []byte("BLMF")
	h, ok := bf.hasher.(PersistentHasher)
	if !ok {
		return nil, ErrorNoPersistentHasher
	}

	s1, s2 := h.Seed()

	return &header{
		Magic:     [4]byte(m),
		Version:   Version1,
		HashID:    uint8(h.ID()),
		Reserved1: [2]byte{0x0, 0x0},
		K:         bf.kSize,
		Reserved2: [4]byte{0x0, 0x0, 0x0, 0x0},
		M:         bf.mSize,
		Seed1:     s1,
		Seed2:     s2,
	}, nil
}

// BloomFilterFromHeader creates the BloomFilter from the header and the data (Bloomfilter bytearray)
// it returns a pointer to the BloomFilter or an error
func BloomFilterFromHeader(h *header, data []byte) (*BloomFilter, error) {
	hid := HashID(h.HashID)
	var hasher Hasher

	switch hid {
	case HashMurmur3x64:
		hasher = NewMurmur3HasherWithSeeds(h.Seed1, h.Seed2)
	default:
		return nil, ErrorNoSupportedHasher
	}

	return &BloomFilter{
		mSize:  h.M,
		kSize:  h.K,
		hasher: hasher,
		filter: data,
		rwMu:   new(sync.RWMutex),
	}, nil
}
