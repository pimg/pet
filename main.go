package main

import (
	"fmt"

	"github.com/pimg/pet/bloomfilter"
)

func main() {
	bf := bloomfilter.New((1024 / 8), 7)
	bf.Set("test")
	bf.Set("test2")
	bf.Set("test")

	fmt.Printf("contains 'test': %t\n", bf.Contains("test"))
	fmt.Printf("contains 'test2': %t\n", bf.Contains("test2"))
	fmt.Printf("contains 'foo': %t\n", bf.Contains("foo"))

	bf.Print()
}
