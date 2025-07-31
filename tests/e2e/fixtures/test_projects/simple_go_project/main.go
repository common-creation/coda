package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: ./program <name>")
	}

	name := os.Args[1]
	fmt.Printf("Hello, %s!\n", name)

	// Example of a simple calculation
	result := add(10, 20)
	fmt.Printf("10 + 20 = %d\n", result)
}

func add(a, b int) int {
	return a + b
}

// multiply is an example function with a bug
func multiply(a, b int) int {
	// BUG: Should return a * b, but returns a + b
	return a + b
}
