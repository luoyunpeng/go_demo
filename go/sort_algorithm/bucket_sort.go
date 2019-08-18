package main

import "fmt"

func bucketSort(a []int, max int) {
	if a == nil || max < 1 {
		return
	}
	// 创建一个容量为max的数组buckets，并且将buckets中的所有数据都初始化为0。
	buckets := make([]int, max)

	// 1. 计数
	for i := 0; i < len(a); i++ {
		buckets[a[i]]++
	}

	// 2. 排序
	for i, j := 0, 0; i < max; i++ {
		for buckets[i] > 0 {
			a[j] = i
			j++
			buckets[i] = buckets[i] - 1
		}
	}
}

func main() {
	a := []int{8, 2, 3, 4, 3, 6, 6, 3, 9}
	fmt.Println("before sort: ")
	fmt.Println(a)

	bucketSort(a, 10)
	fmt.Println("after sort: ")
	fmt.Println(a)
}
