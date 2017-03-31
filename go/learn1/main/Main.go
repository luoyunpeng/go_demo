package main

import (
	"fmt"
)

func main() {
	Print(6)
}

func Print(a int) {
	if a > 5 {
		fmt.Print(":: ", a)
	}
	fmt.Print("let's do it ")
}
