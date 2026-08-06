package main

import (
	"fmt"

	"github.com/pimg/pet/bloomfilter"
)

func main() {
	// bf := bloomfilter.NewFromEstimate(100, 0.01)
	bf, _ := bloomfilter.New(bloomfilter.WithPresetSize(bloomfilter.Small()))
	bf.Add("test")
	bf.Add("test2")
	bf.Add("test")

	fmt.Printf("contains 'test': %t\n", bf.Contains("test"))
	fmt.Printf("contains 'test2': %t\n", bf.Contains("test2"))
	fmt.Printf("contains 'foo': %t\n", bf.Contains("foo"))

	// bf.Print()
}
