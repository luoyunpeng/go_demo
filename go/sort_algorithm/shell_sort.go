package main

import "fmt"

func main() {
	s := []int{8, 5, 4, 7, 2, 6, 1, 2, 15}
	fmt.Println("before sort:")
	fmt.Println(s)
	shellSort(s)
	fmt.Println("after shell  sort:")
	fmt.Println(s)
}

func shellSort(arr []int) {
	length := len(arr)
	gap := 1
	for gap < gap/3 {
		gap = gap*3 + 1
	}
	for gap > 0 {
		for i := gap; i < length; i++ {
			temp := arr[i]
			j := i - gap
			for j >= 0 && arr[j] > temp {
				arr[j+gap] = arr[j]
				j -= gap
			}
			arr[j+gap] = temp
		}
		gap = gap / 3
	}
}
