package main

import "fmt"

func main() {
	s := []int{8, 5, 4, 7, 2, 6, 1, 2, 15}
	fmt.Println("before sort:")
	fmt.Println(s)
	selectSortOptmize(s)
	fmt.Println("after sort:")
	fmt.Println(s)
}

func selectSort(s []int) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// just select the value and not swap in 'for' statement
func selectSortOptmize(s []int) {
	index := 0
	num := s[0]

	for i := 0; i < len(s)-1; i++ {
		index = i
		num = s[i]
		for j := i + 1; j < len(s); j++ {
			if num > s[j] {
				index = j
				num = s[j]
			}
		}

		if index != i {
			s[i], s[index] = s[index], s[i]
		}
		//fmt.Println(s)
	}
}
