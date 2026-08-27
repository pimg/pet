package bloomfilter

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
)

func Test_NoPersistentHasher(t *testing.T) {
	bf, _ := New(WithSize(Small()))
	var rw bytes.Buffer
	err := bf.Encode(&rw)
	if !errors.Is(err, ErrorNoPersistentHasher) {
		t.Fail()
	}
}

func Test_encodeDecode(t *testing.T) {
	bf, _ := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))
	bf.Add("test1")
	bf.Add("test2")
	bf.Add("test3")

	var rw bytes.Buffer

	err := bf.Encode(&rw)
	if err != nil {
		t.Fatalf("encode returned error: %v", err)
	}

	bf.filter = nil // hard reset the filter to ensure bf.filter is set in Decode

	err = bf.Decode(&rw)
	if err != nil {
		t.Fatalf("decode returned error: %v", err)
	}

	if !bf.Contains("test1") {
		t.Fatal("decoding failed bloomfilter should contain 'test1'")
	}

	if bf.Contains("foo") {
		t.Fatal("decoding failed bloomfilter should not contain 'foo'")
	}
}

func Test_encodeDecodeCompression(t *testing.T) {
	bf, _ := New(WithSize(Small()), WithCompression(), WithHasher(NewMurmur3Hasher()))
	bf.Add("test1")
	bf.Add("test2")
	bf.Add("test3")

	var rw bytes.Buffer

	err := bf.Encode(&rw)
	if err != nil {
		t.Fatalf("encode returned error: %v", err)
	}

	bf.filter = nil // hard reset the filter to ensure bf.filter is set in Decode

	err = bf.Decode(&rw)
	if err != nil {
		t.Fatalf("decode returned error: %v", err)
	}

	if !bf.Contains("test1") {
		t.Fatal("decoding failed bloomfilter should contain 'test1'")
	}

	if bf.Contains("foo") {
		t.Fatal("decoding failed bloomfilter should not contain 'foo'")
	}
}

func Test_persistenceWithFile(t *testing.T) {
	bf, _ := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))
	bf.Add("test1")
	bf.Add("test2")
	bf.Add("test3")

	testFile := os.TempDir() + "/test.bin"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file test.bin: %v", err)
	}
	err = bf.Encode(file)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	bf.filter = nil // hard reset the filter to ensure bf.filter is set in Decode

	readFile, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to read test.bin: %v", err)
	}
	defer func() {
		err = readFile.Close()
		if err != nil {
			t.Fatalf("failed to close file reader: %v", err)
		}
	}()

	err = bf.Decode(readFile)
	if err != nil {
		t.Fatalf("decode returned error: %v", err)
	}

	if !bf.Contains("test1") {
		t.Fatal("decoding failed bloomfilter should contain 'test1'")
	}

	if bf.Contains("foo") {
		t.Fatal("decoding failed bloomfilter should not contain 'foo'")
	}

	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}
}

func Test_persistenceWithFileCompression(t *testing.T) {
	bf, _ := New(WithSize(Small()), WithCompression(), WithHasher(NewMurmur3Hasher()))
	bf.Add("test1")
	bf.Add("test2")
	bf.Add("test3")

	testFile := os.TempDir() + "/test.bin"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file test.bin: %v", err)
	}
	err = bf.Encode(file)
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}

	err = file.Close()
	if err != nil {
		t.Fatalf("failed to close file: %v", err)
	}

	bf.filter = nil // hard reset the filter to ensure bf.filter is set in Decode

	readFile, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to read test.bin: %v", err)
	}
	defer func() {
		err = readFile.Close()
		if err != nil {
			t.Fatalf("failed to close file: %v", err)
		}
	}()

	err = bf.Decode(readFile)
	if err != nil {
		t.Fatalf("decode returned error: %v", err)
	}

	if !bf.Contains("test1") {
		t.Fatal("decoding failed bloomfilter should contain 'test1'")
	}

	if bf.Contains("foo") {
		t.Fatal("decoding failed bloomfilter should not contain 'foo'")
	}

	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}
}

func Test_concurrentWrites(t *testing.T) {
	bf, _ := New(WithSize(Small()), WithHasher(NewMurmur3Hasher()))

	rounds := 5000
	wg := sync.WaitGroup{}
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
