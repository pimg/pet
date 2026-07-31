package bloomfilter

import (
	"fmt"
	"math"
	"testing"
)

func Test_EmptyFalsePositive(t *testing.T) {
	bf := New()

	for i := range 3000 {
		if bf.Contains(fmt.Sprintf("miss-%d", i)) {
			t.Errorf("found a false positive 'miss-%d'", i)
		}
	}
}

func Test_200ItemsFalsePositive(t *testing.T) {
	const n = 200
	const queries = 3000

	bf := New()
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
	bf := New()
	testData := "this is a test"

	bf.Set(testData)

	if !bf.Contains(testData) {
		t.Fatalf("Bloomfilter should contain: %s", testData)
	}
}

func Test_EmptySetAndDoesNotContain(t *testing.T) {
	bf := New()
	testData := "I am not set in the Bloom filter"

	if bf.Contains(testData) {
		t.Fatalf("Bloomfilter should not contain: %s", testData)
	}
}

func Test_Inclusion(t *testing.T) {
	bf := New()

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
