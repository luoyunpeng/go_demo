package main

import "fmt"

func main() {
	s := []int{8, 5, 4, 7, 2, 6, 1, 2, 15}
	fmt.Println("before sort:")
	fmt.Println(s)
	insertSort(s)
	fmt.Println("after insert sort:")
	fmt.Println(s)
}

/*
把arr,分为连续的两部分, 第一部分为已排序, 第二未排序
1, 选取index=0,为第一部分, 第二部分为index=1~len(arr)-1
2, 从index=1开始,将数据与第一部分比较, 如果小于则插入,直到与第一部分数据组成有序的数组
3, index+1，重复2，
# 若index 大于 preIndex则插入到preIndex之前(插入为移动, current记录当前比较数据index, arr[index] = arr[preIndex])
*/
func insertSort(arr []int) {
	arrLen := len(arr)
	for i := 1; i < arrLen; i++ {
		preIndex := i - 1
		current := arr[i]
		for preIndex >= 0 && current < arr[preIndex] {
			arr[preIndex+1] = arr[preIndex] // current = preIndex, or:  arr[preIndex],arr[preIndex+1]  =  arr[preIndex+1], arr[preIndex]
			preIndex--
		}
		if preIndex+1 != i {
			arr[preIndex+1] = current // if
		}
	}
}
