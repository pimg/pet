package bloomfilter

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"testing"
)

const filterSize = 128
const kSize = 7

func Test_EmptyFalsePositive(t *testing.T) {
	bf, err := New(WithSize(filterSize, kSize))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	for i := range 3000 {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			t.Errorf("found a false positive 'miss-%d'", i)
		}
	}
}

func Test_200ItemsFalsePositive(t *testing.T) {
	const n = 200
	const queries = 3000

	bf, err := New(WithSize(filterSize, kSize))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	for i := range n {
		bf.Add(fmt.Sprintf("entry-%d", i))
	}

	fpCount := 0
	for i := range queries {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			fpCount++
		}
	}

	// Theoretical false positive rate: (1 - e^(-kn/m))^k
	expected := math.Pow(1-math.Exp(-float64(kSize*n)/float64(filterSize)), float64(kSize))
	measured := float64(fpCount) / float64(queries)

	const tolerance = 0.04
	if math.Abs(measured-expected) > tolerance {
		t.Fatalf("false positive rate %.2f%% (%d/%d), expected %.2f%% ± %.0f%%",
			measured*100, fpCount, queries, expected*100, tolerance*100)
	}
}

func Test_SetAndContains(t *testing.T) {
	bf, err := New(WithSize(filterSize, kSize))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := "this is a test"

	bf.Add(testData)

	if !bf.Contains(testData) {
		t.Fatalf("Bloomfilter should contain: %s", testData)
	}
}

func Test_EmptySetAndDoesNotContain(t *testing.T) {
	bf, err := New(WithSize(filterSize, kSize))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := "I am not set in the Bloom filter"

	if bf.Contains(testData) {
		t.Fatalf("Bloomfilter should not contain: %s", testData)
	}
}

func Test_Inclusion(t *testing.T) {
	bf, err := New(WithPresetSize(Parameters{filterSize, kSize}))
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	// inserting 200 items in the Bloomfilter
	for i := range 200 {
		bf.Add(fmt.Sprintf("entry-%d", i))
	}

	for i := range 200 {
		if !bf.Contains(fmt.Sprintf("entry-%d", i)) {
			t.Fatalf("Bloom filter should contain: entry-%d", i)
		}
	}
}

func Test_NewWithEstimates(t *testing.T) {
	bf, err := NewFromEstimate(1000, 0.01)
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	if bf.mSize != 9586 {
		t.Fatal("wrong filtersize for estimate")
	}

	if bf.kSize != 7 {
		t.Fatal("wrong kSize for estimate")
	}
}

func Test_WithSmall(t *testing.T) {
	bf, err := New(WithPresetSize(Small()))
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	if bf.mSize != 9586 {
		t.Fatalf("filter size for small should be 9686, got: %d", bf.mSize)
	}

	if bf.kSize != 7 {
		t.Fatalf("kSize for small should be 7, got: %d", bf.kSize)
	}

	bf.Add("test")
	if !bf.Contains("test") {
		t.Fatal("bloomfilter should contain 'test'")
	}

	if bf.Contains("unknown") {
		t.Fatal("bloomfilter should not contain 'unknown'")
	}
}

func Test_WithMedium(t *testing.T) {
	bf, err := New(WithPresetSize(Medium()))
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	if bf.mSize != 958506 {
		t.Fatalf("filter size for small should be 958506, got: %d", bf.mSize)
	}

	if bf.kSize != 7 {
		t.Fatalf("kSize for small should be 7, got: %d", bf.kSize)
	}

	bf.Add("test")
	if !bf.Contains("test") {
		t.Fatal("bloomfilter should contain 'test'")
	}

	if bf.Contains("unknown") {
		t.Fatal("bloomfilter should not contain 'unknown'")
	}
}

func Test_WithLarge(t *testing.T) {
	bf, err := New(WithPresetSize(Large()))
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	if bf.mSize != 14377588 {
		t.Fatalf("filter size for small should be 14377588, got: %d", bf.mSize)
	}

	if bf.kSize != 10 {
		t.Fatalf("kSize for small should be 10, got: %d", bf.kSize)
	}

	bf.Add("test")
	if !bf.Contains("test") {
		t.Fatal("bloomfilter should contain 'test'")
	}

	if bf.Contains("unknown") {
		t.Fatal("bloomfilter should not contain 'unknown'")
	}
}

func Test_FalsePositiveRate(t *testing.T) {
	const n = 1000
	const p = 0.01
	const queries = 10000

	bf, err := NewFromEstimate(n, p)
	if err != nil {
		t.Fatal("failed to initialize bloomfilter")
	}

	for i := range n {
		bf.Add(fmt.Sprintf("entry-%d", i))
	}

	fpCount := 0
	for i := range queries {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			fpCount++
		}
	}

	// Theoretical false positive rate: (1 - e^(-kn/m))^k
	// Uses the filter's actual m and k, which differ from p due to rounding.
	expected := math.Pow(1-math.Exp(-float64(bf.kSize*n)/float64(bf.mSize)), float64(bf.kSize))
	measured := float64(fpCount) / float64(queries)

	sigma := math.Sqrt(expected * (1 - expected) / float64(queries))
	tolerance := 5 * sigma

	if math.Abs(measured-expected) > tolerance {
		t.Fatalf("false positive rate %.3f%% (%d/%d), expected %.3f%% ± %.3f%% (m=%d, k=%d)",
			measured*100, fpCount, queries, expected*100, tolerance*100, bf.mSize, bf.kSize)
	}
}

func Test_invalidBloomfilter(t *testing.T) {
	_, err := New()
	if err == nil {
		t.Fatal("a bloomfilter without parameters should not be initialized")
	}
}

func Test_missingKsize(t *testing.T) {
	_, err := New(WithPresetSize(Parameters{M: 10}))
	if err == nil {
		t.Fatal("a bloomfilter without kSize should not be initialized")
	}

	if err.Error() != "bloomfilter initialized without kSize (k)" {
		t.Fatalf("New returned an invalid error: %v", err)
	}
}

func Test_missingM(t *testing.T) {
	_, err := New(WithPresetSize(Parameters{K: 10}))
	if err == nil {
		t.Fatal("a bloomfilter without filterSize (m) should not be initialized")
	}

	if err.Error() != "bloomfilter initialized without filtersize (m)" {
		t.Fatalf("New returned an invalid error: %v", err)
	}
}

func Test_encodeDecode(t *testing.T) {
	bf, _ := New(WithPresetSize(Small()))
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
	bf, _ := New(WithPresetSize(Small()))
	bf.Add("test1")
	bf.Add("test2")
	bf.Add("test3")

	testFile := os.TempDir() + "/test.bin"
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file test.bin: %v", err)
	}
	bf.Encode(file)
	file.Close()

	bf.filter = nil // hard reset the filter to ensure bf.filter is set in Decode

	readFile, err := os.Open(testFile)
	if err != nil {
		t.Fatalf("failed to read test.bin: %v", err)
	}
	defer readFile.Close()

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

	os.Remove(testFile)
}
