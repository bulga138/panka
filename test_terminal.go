package main

import (
	"fmt"
	"time"
)

func test() {
	fmt.Print("\x1b[2J") // Clear screen
	fmt.Print("\x1b[H")  // Home
	fmt.Print("   1 Hello World\r\n")
	fmt.Print("   2 ~\r\n")
	fmt.Print("   3 ~\r\n")
	fmt.Print("   4 ~\r\n")
	fmt.Print("\x1b[1;10H") // Position cursor
	fmt.Printf("Cursor should be at line 1, column 10")

	time.Sleep(5 * time.Second)
	fmt.Print("\x1b[2J\x1b[H") // Clean exit
}
