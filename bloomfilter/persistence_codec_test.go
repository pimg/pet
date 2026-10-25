package bloomfilter

import (
	"errors"
	"testing"
)

func TestHeaderInvalidMagic(t *testing.T) {
	h := header{
		Magic: [4]byte{'b', 'l'},
	}

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderMagic) {
		t.Fatalf("expected invalid magic header, got: %v", err)
	}
}

func TestHeaderInvalidVersion(t *testing.T) {
	h := createValidHeader()
	h.Version = 2

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderVersion) {
		t.Fatalf("expected invalid version, got: %v", err)
	}
}

func TestHeaderInvalidHashID(t *testing.T) {
	h := createValidHeader()
	h.HashID = 4

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderHashID) {
		t.Fatalf("expected invalid header, got: %v", err)
	}
}

func TestHeaderInvalidReserved(t *testing.T) {
	h := createValidHeader()
	h.Reserved1 = [2]byte{'f', 0x00}

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderReserved) {
		t.Fatalf("expected invalid reserved, got: %v", err)
	}
}

func TestHeaderInvalidReserved2(t *testing.T) {
	h := createValidHeader()
	h.Reserved2 = [4]byte{'f', 0x00}

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderReserved) {
		t.Fatalf("expected invalid reserved, got: %v", err)
	}
}

func TestHeaderInvalidK(t *testing.T) {
	h := createValidHeader()
	h.K = 1

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderKM) {
		t.Fatalf("expected invalid K or M, got: %v", err)
	}
}

func TestHeaderInvalidM(t *testing.T) {
	h := createValidHeader()
	h.M = 1

	err := h.Validate()
	if !errors.Is(err, ErrorInvalidHeaderKM) {
		t.Fatalf("expected invalid K or M, got: %v", err)
	}
}

func TestHeaderValid(t *testing.T) {
	h := createValidHeader()
	if err := h.Validate(); err != nil {
		t.Fatalf("expected valid header got error: %v", err)
	}
}

func createValidHeader() header {
	bf, _ := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))

	header, _ := NewHeader(bf)
	return *header
}
