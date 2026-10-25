package bloomfilter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// newPersistentFilter builds a Small filter with a persistent hasher and some content.
func newPersistentFilter(t *testing.T) *BloomFilter {
	t.Helper()
	bf, err := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	for i := range 100 {
		bf.Add(fmt.Sprintf("entry-%d", i))
	}
	return bf
}

// assertSameAnswers checks that two filters answer identically,
// for both inserted items and items that were never inserted.
func assertSameAnswers(t *testing.T, want, got *BloomFilter) {
	t.Helper()
	for i := range 100 {
		v := fmt.Sprintf("entry-%d", i)
		if !got.Contains(v) {
			t.Fatalf("loaded filter should contain %q", v)
		}
	}
	for i := range 1000 {
		v := fmt.Sprintf("miss-%d", i)
		if want.Contains(v) != got.Contains(v) {
			t.Fatalf("filters disagree on %q", v)
		}
	}
}

func Test_NoPersistentHasher(t *testing.T) {
	bf, err := New(WithSize(Small()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if _, err := bf.MarshalBinary(); !errors.Is(err, ErrorInvalidBloomFilter) {
		t.Fatalf("MarshalBinary: expected ErrorInvalidBloomFilter, got %v", err)
	}

	var buf bytes.Buffer
	if _, err := bf.WriteTo(&buf); !errors.Is(err, ErrorInvalidBloomFilter) {
		t.Fatalf("WriteTo: expected ErrorInvalidBloomFilter, got %v", err)
	}
	if buf.Len() != 0 {
		t.Fatalf("WriteTo wrote %d bytes despite failing", buf.Len())
	}
}

func Test_MarshalUnmarshalRoundTrip(t *testing.T) {
	bf := newPersistentFilter(t)

	data, err := bf.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary returned error: %v", err)
	}

	var loaded BloomFilter // zero value: everything must come from data
	if err := loaded.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary returned error: %v", err)
	}

	assertSameAnswers(t, bf, &loaded)
}

func Test_MarshalIsCanonical(t *testing.T) {
	bf := newPersistentFilter(t)

	first, err := bf.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary returned error: %v", err)
	}

	var loaded BloomFilter
	if err := loaded.UnmarshalBinary(first); err != nil {
		t.Fatalf("UnmarshalBinary returned error: %v", err)
	}

	second, err := loaded.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary of loaded filter returned error: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("re-marshalled filter differs from the original bytes")
	}
}

func Test_WriteToReadFromRoundTrip(t *testing.T) {
	bf := newPersistentFilter(t)

	var buf bytes.Buffer
	written, err := bf.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo returned error: %v", err)
	}
	if written != int64(buf.Len()) {
		t.Fatalf("WriteTo reported %d bytes, buffer holds %d", written, buf.Len())
	}

	var loaded BloomFilter
	read, err := loaded.ReadFrom(&buf)
	if err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if read != written {
		t.Fatalf("ReadFrom read %d bytes, WriteTo wrote %d", read, written)
	}

	assertSameAnswers(t, bf, &loaded)
}

func Test_ReadFromStopsAtEndOfFilter(t *testing.T) {
	bf := newPersistentFilter(t)

	var buf bytes.Buffer
	if _, err := bf.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo returned error: %v", err)
	}
	trailer := []byte("next record")
	buf.Write(trailer)

	var loaded BloomFilter
	if _, err := loaded.ReadFrom(&buf); err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), trailer) {
		t.Fatalf("ReadFrom consumed data after the filter; remaining: %q", buf.Bytes())
	}
}

func Test_PersistenceWithFile(t *testing.T) {
	bf := newPersistentFilter(t)
	path := filepath.Join(t.TempDir(), "filter.bin") // removed automatically

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := bf.WriteTo(f); err != nil {
		err := f.Close()
		if err != nil {
			t.Log(err)
		}
		t.Fatalf("WriteTo returned error: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close after write: %v", err)
	}

	f, err = os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			t.Log(err)
		}
	}()

	var loaded BloomFilter
	if _, err := loaded.ReadFrom(f); err != nil {
		t.Fatalf("ReadFrom returned error: %v", err)
	}

	assertSameAnswers(t, bf, &loaded)
}

func Test_UnmarshalRejectsCorruption(t *testing.T) {
	bf := newPersistentFilter(t)
	data, err := bf.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary returned error: %v", err)
	}

	// One flipped bit in the header, in the bit array, and in the CRC itself.
	for _, pos := range []int{5, 40, len(data) / 2, len(data) - 1} {
		corrupt := bytes.Clone(data)
		corrupt[pos] ^= 0x01

		var loaded BloomFilter
		if err := loaded.UnmarshalBinary(corrupt); !errors.Is(err, ErrorChecksum) {
			t.Errorf("flip at byte %d: expected ErrChecksum, got %v", pos, err)
		}
	}
}

func Test_UnmarshalRejectsTruncation(t *testing.T) {
	bf := newPersistentFilter(t)
	data, err := bf.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary returned error: %v", err)
	}

	for _, n := range []int{0, 10, 43, len(data) - 1} {
		var loaded BloomFilter
		if err := loaded.UnmarshalBinary(data[:n]); err == nil {
			t.Errorf("UnmarshalBinary accepted %d of %d bytes", n, len(data))
		}
	}
}

func Test_ReadFromRejectsTruncation(t *testing.T) {
	bf := newPersistentFilter(t)
	data, err := bf.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary returned error: %v", err)
	}

	var loaded BloomFilter
	_, err = loaded.ReadFrom(bytes.NewReader(data[:len(data)-1]))

	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected ErrUnexpectedEOF:', got %v", err)
	}
}

func Test_ConcurrentWrites(t *testing.T) {
	bf, err := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	const rounds = 5000
	var wg sync.WaitGroup
	for i := range rounds {
		wg.Go(func() {
			bf.Add(fmt.Sprintf("test_%d", i))
		})
	}
	wg.Wait()

	for i := range rounds {
		if !bf.Contains(fmt.Sprintf("test_%d", i)) {
			t.Errorf("bloomfilter did not contain: test_%d", i)
		}
	}
}
