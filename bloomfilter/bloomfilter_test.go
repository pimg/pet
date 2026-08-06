package bloomfilter

import (
	"fmt"
	"math"
	"testing"
)

const filterSize = 128
const kSize = 7

func Test_EmptyFalsePositive(t *testing.T) {
	bf := new(filterSize, kSize)

	for i := range 3000 {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			t.Errorf("found a false positive 'miss-%d'", i)
		}
	}
}

func Test_200ItemsFalsePositive(t *testing.T) {
	const n = 200
	const queries = 3000

	bf := new(filterSize, kSize)
	for i := range n {
		bf.Set(fmt.Sprintf("entry-%d", i))
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
	bf := new(filterSize, kSize)
	testData := "this is a test"

	bf.Set(testData)

	if !bf.Contains(testData) {
		t.Fatalf("Bloomfilter should contain: %s", testData)
	}
}

func Test_EmptySetAndDoesNotContain(t *testing.T) {
	bf := new(filterSize, kSize)
	testData := "I am not set in the Bloom filter"

	if bf.Contains(testData) {
		t.Fatalf("Bloomfilter should not contain: %s", testData)
	}
}

func Test_Inclusion(t *testing.T) {
	bf := new(filterSize, kSize)

	// inserting 200 items in the Bloomfilter
	for i := range 200 {
		bf.Set(fmt.Sprintf("entry-%d", i))
	}

	for i := range 200 {
		if !bf.Contains(fmt.Sprintf("entry-%d", i)) {
			t.Fatalf("Bloom filter should contain: entry-%d", i)
		}
	}
}

func Test_NewWithEstimates(t *testing.T) {
	bf := NewFromEstimate(1000, 0.01)
	if bf.filterSize != 9585 {
		t.Fatal("wrong filtersize for estimate")
	}

	if bf.kSize != 7 {
		t.Fatal("wrong kSize for estimate")
	}
}

func Test_WithSmall(t *testing.T) {
	bf := New(Small())
	if bf.filterSize != 9586 {
		t.Fatalf("filter size for small should be 9686, got: %d", bf.filterSize)
	}

	if bf.kSize != 7 {
		t.Fatalf("kSize for small should be 7, got: %d", bf.kSize)
	}

	bf.Set("test")
	if !bf.Contains("test") {
		t.Fatal("bloomfilter should contain 'test'")
	}

	if bf.Contains("unknown") {
		t.Fatal("bloomfilter should not contain 'unknown'")
	}
}

func Test_WithMedium(t *testing.T) {
	bf := New(Medium())
	if bf.filterSize != 958506 {
		t.Fatalf("filter size for small should be 958506, got: %d", bf.filterSize)
	}

	if bf.kSize != 7 {
		t.Fatalf("kSize for small should be 7, got: %d", bf.kSize)
	}

	bf.Set("test")
	if !bf.Contains("test") {
		t.Fatal("bloomfilter should contain 'test'")
	}

	if bf.Contains("unknown") {
		t.Fatal("bloomfilter should not contain 'unknown'")
	}
}

func Test_WithLarge(t *testing.T) {
	bf := New(Large())
	if bf.filterSize != 14377588 {
		t.Fatalf("filter size for small should be 14377588, got: %d", bf.filterSize)
	}

	if bf.kSize != 10 {
		t.Fatalf("kSize for small should be 10, got: %d", bf.kSize)
	}

	bf.Set("test")
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

	bf := NewFromEstimate(n, p)

	for i := range n {
		bf.Set(fmt.Sprintf("entry-%d", i))
	}

	fpCount := 0
	for i := range queries {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			fpCount++
		}
	}

	// Theoretical false positive rate: (1 - e^(-kn/m))^k
	// Uses the filter's actual m and k, which differ from p due to rounding.
	expected := math.Pow(1-math.Exp(-float64(bf.kSize*n)/float64(bf.filterSize)), float64(bf.kSize))
	measured := float64(fpCount) / float64(queries)

	sigma := math.Sqrt(expected * (1 - expected) / float64(queries))
	tolerance := 5 * sigma

	if math.Abs(measured-expected) > tolerance {
		t.Fatalf("false positive rate %.3f%% (%d/%d), expected %.3f%% ± %.3f%% (m=%d, k=%d)",
			measured*100, fpCount, queries, expected*100, tolerance*100, bf.filterSize, bf.kSize)
	}
}
