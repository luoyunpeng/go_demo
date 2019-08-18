package main

import "fmt"

func main() {
	s := []int{8, 5, 4, 7, 2, 6, 1, 2, 15}
	fmt.Println("before sort:")
	fmt.Println(s)
	bubble(s)
	fmt.Println("after sort:")
	fmt.Println(s)
}

func bubble(s []int) {
	sLen := len(s)
	for i := 0; i < sLen-1; i++ {
		for j := 0; j < sLen-1-i; j++ {
			if s[j] > s[j+1] {
				s[j], s[j+1] = s[j+1], s[j]
			}
		}
	}
}
