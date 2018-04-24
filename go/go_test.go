package main

import (
	"fmt"
	"testing"
)

// Test for defer execution
// The result is locate, 4, 3, 2, 1, 0, true
func TestDefer(t *testing.T) {
	fmt.Println(testDefer())
}

func testDefer() bool {
	for i := 0; i < 5; i++ {
		defer fmt.Println(i)
	}
	return testDeferP()
}

func testDeferP() bool {
	fmt.Println("locate")
	return true
}
