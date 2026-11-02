package bloomfilter

import (
	"fmt"
	"math"
	"testing"
)

const filterSize = 128
const kSize = 7

func Test_EmptyFalsePositive(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
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

	bf, err := New(WithSize(Parameters{filterSize, kSize}))
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
	measured := bf.CurrentFalsePositiveRate()

	const tolerance = 0.04
	if math.Abs(measured-expected) > tolerance {
		t.Fatalf("false positive rate %.2f%% (%d/%d), expected %.2f%% ± %.0f%%",
			measured*100, fpCount, queries, expected*100, tolerance*100)
	}
}

func Test_SetAndContains(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := "this is a test"

	bf.Add(testData)

	if !bf.Contains(testData) {
		t.Fatalf("Bloomfilter should contain: %s", testData)
	}
}

func Test_SetAndContainsBytes(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := []byte("this is a test")

	bf.AddBytes(testData)

	if !bf.ContainsBytes(testData) {
		t.Fatalf("Bloomfilter should contain: %s", testData)
	}
}

func Test_EmptySetAndDoesNotContain(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := "I am not set in the Bloom filter"

	if bf.Contains(testData) {
		t.Fatalf("Bloomfilter should not contain: %s", testData)
	}
}

func Test_EmptySetAndDoesNotContainBytes(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
	if err != nil {
		t.Fatal("failed to initialize bloomfitler")
	}

	testData := []byte("I am not set in the Bloom filter")

	if bf.ContainsBytes(testData) {
		t.Fatalf("Bloomfilter should not contain: %s", testData)
	}
}

func Test_Inclusion(t *testing.T) {
	bf, err := New(WithSize(Parameters{filterSize, kSize}))
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
	bf, err := New(WithSize(Small()))
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
	bf, err := New(WithSize(Medium()))
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
	bf, err := New(WithSize(Large()))
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
	_, err := New(WithSize(Parameters{M: 10}))
	if err == nil {
		t.Fatal("a bloomfilter without kSize should not be initialized")
	}

	if err.Error() != "bloomfilter initialized without kSize (k)" {
		t.Fatalf("New returned an invalid error: %v", err)
	}
}

func Test_missingM(t *testing.T) {
	_, err := New(WithSize(Parameters{K: 10}))
	if err == nil {
		t.Fatal("a bloomfilter without filterSize (m) should not be initialized")
	}

	if err.Error() != "bloomfilter initialized without filtersize (m)" {
		t.Fatalf("New returned an invalid error: %v", err)
	}
}

// Expected values are computed independently of the implementation, from:
//
//	e = lambda + log2(clientItems)
//	m = ceil(serverItems * e / ln2)
//	k = ceil(e)
func Test_PSIParameters(t *testing.T) {
	tests := []struct {
		name                   string
		server, client, lambda int
		wantM, wantK           uint64
	}{
		{"design example: 2^20 vs 2^20, lambda 40", 1 << 20, 1 << 20, 40, 90766524, 60},
		{"single client item: e equals lambda", 1000, 1, 40, 57708, 40},
		{"non power of two client count", 1000, 1000, 40, 72086, 50},
		{"k rounds up, not to nearest (e = 12.32)", 100, 5, 10, 1778, 13},
		{"smallest valid input", 1, 1, 1, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, k, err := psiParameters(tt.server, tt.client, tt.lambda)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if m != tt.wantM {
				t.Errorf("m = %d, want %d", m, tt.wantM)
			}
			if k != tt.wantK {
				t.Errorf("k = %d, want %d", k, tt.wantK)
			}
		})
	}
}

func Test_PSIParametersRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name                   string
		server, client, lambda int
	}{
		{"zero server items", 0, 1000, 40},
		{"negative server items", -1, 1000, 40},
		{"zero client items", 1000, 0, 40},
		{"negative client items", 1000, -1, 40},
		{"zero lambda", 1000, 1000, 0},
		{"negative lambda", 1000, 1000, -5},
		{"lambda too large", 1000, 1000, 1000},
		{"filter too large to load", math.MaxInt, 1000, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := psiParameters(tt.server, tt.client, tt.lambda); err == nil {
				t.Errorf("expected an error for server=%d client=%d lambda=%d",
					tt.server, tt.client, tt.lambda)
			}
		})
	}
}

// The option must pass validation errors through New instead of
// producing a filter with zero or nonsensical parameters.
func Test_WithPSISizeReturnsError(t *testing.T) {
	bf, err := New(ForPSI(0, 1000, 40), WithHasher(NewMurmur3Hasher()))
	if err == nil {
		t.Fatal("expected New to return an error")
	}
	if bf != nil {
		t.Fatal("expected a nil filter on error")
	}
}

// Behavioral check with a small lambda, so false positives are frequent
// enough to measure. If the sizing is right here, it is right at lambda 40.
//
// With clientItems = 1 the per-query target is exactly 2^-lambda.
func Test_PSISizingMeetsTargetRate(t *testing.T) {
	const (
		serverItems = 10000
		lambda      = 8
		queries     = 200000
	)

	bf, err := New(ForPSI(serverItems, 1, lambda), WithHasher(NewMurmur3Hasher()))
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	for i := range serverItems {
		bf.Add(fmt.Sprintf("server-%d", i))
	}

	fpCount := 0
	for i := range queries {
		if bf.Contains(fmt.Sprintf("query-%d", i)) {
			fpCount++
		}
	}

	target := math.Pow(2, -lambda)
	measured := float64(fpCount) / float64(queries)

	// The rounding (ceil of m and k) should never make the filter worse than
	// the target, so check the theoretical rate of the actual m and k first.
	n := float64(serverItems)
	k := float64(bf.kSize)
	m := float64(bf.mSize)
	expected := math.Pow(1-math.Exp(-k*n/m), k)
	if expected > target {
		t.Fatalf("theoretical rate %.5f%% exceeds target %.5f%% (m=%d, k=%d)",
			expected*100, target*100, bf.mSize, bf.kSize)
	}

	sigma := math.Sqrt(expected * (1 - expected) / float64(queries))
	if math.Abs(measured-expected) > 5*sigma {
		t.Fatalf("false positive rate %.4f%% (%d/%d), expected %.4f%% ± %.4f%% (m=%d, k=%d)",
			measured*100, fpCount, queries, expected*100, 5*sigma*100, bf.mSize, bf.kSize)
	}
}
