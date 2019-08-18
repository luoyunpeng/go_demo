package main

import (
	"fmt"
)

func main() {
	s := []int{1, 7, 6, 5, 4, 2, 3}
	fmt.Println("before sort:")
	fmt.Println(s)
	Quick2Sort(s)
	fmt.Println("after sort:")
	fmt.Println(s)

	s1 := []int{1, 7, 6, 5, 4, 2, 3}
	fmt.Println("before qsort:")
	fmt.Println(s1)
	qSort(s1)
	fmt.Println("after qsort:")
	fmt.Println(s1)
}

func Quick2Sort(values []int) {
	if len(values) <= 1 {
		return
	}
	mid, i := values[0], 1
	head, tail := 0, len(values)-1
	for head < tail {
		fmt.Println(values)
		if values[i] > mid {
			values[i], values[tail] = values[tail], values[i]
			tail--
		} else {
			values[i], values[head] = values[head], values[i]
			head++
			i++
		}
	}

	values[head] = mid
	fmt.Println("first sort done: ", values)
	Quick2Sort(values[:head])
	Quick2Sort(values[head+1:])
}

func qSort(a []int) {
	if len(a) < 2 {
		return
	}

	q := a[0]
	low, high := 0, len(a)-1

	for low < high {
		for low < high && a[high] >= q {
			high--
		}
		if low < high {
			a[low] = a[high]
			low++
		}

		for low < high && a[low] < q {
			low++
		}
		if low < high {
			a[high] = a[low]
			high--
		}
	}

	a[low] = q
	qSort(a[:low])
	qSort(a[low+1:])
}
