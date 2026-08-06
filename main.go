package main

import (
	"fmt"

	"github.com/pimg/pet/bloomfilter"
)

func main() {
	// bf := bloomfilter.NewFromEstimate(100, 0.01)
	bf, _ := bloomfilter.New(bloomfilter.WithPresetSize(bloomfilter.Small()))
	bf.Set("test")
	bf.Set("test2")
	bf.Set("test")

	fmt.Printf("contains 'test': %t\n", bf.Contains("test"))
	fmt.Printf("contains 'test2': %t\n", bf.Contains("test2"))
	fmt.Printf("contains 'foo': %t\n", bf.Contains("foo"))

	// bf.Print()
}
