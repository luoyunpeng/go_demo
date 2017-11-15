package main

import (
	"fmt"
	"sync"
	"time"
)

var waitGroup sync.WaitGroup

func main() {
	TestWait()
}

func TestWait()  {
	num :=5
	startTime :=time.Now()
	for i:=1;i<=num ;i++  {
		waitGroup.Add(1)
		go  CreateContainer(i)
	}
	waitGroup.Wait()
	fmt.Println("it takes ",time.Now().Sub(startTime)," to create container")
}

func CreateContainer(num int)  {
	fmt.Println("create container: ",num)//start container
	time.Sleep(5*1e9)//assuem starting contianer here
	waitGroup.Done()
}



