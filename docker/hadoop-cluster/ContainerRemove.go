package main

import (
	"github.com/docker/engine-api/client"
	"strconv"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"sync"
	"fmt"

	"time"
)
var waitGroup1 sync.WaitGroup

func main()  {

	startTime := time.Now()
	dockerClient := InitClient1(ip)//ip string value
	for i := 1; i<=30;i++  {
		waitGroup1.Add(1)
		go DeleteContainer(dockerClient,i)
	}
	waitGroup1.Wait()
	fmt.Println("delete done, takes ",time.Now().Sub(startTime)," seconds")
}

func InitClient1(ip string) *client.Client  {
	//defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tcp://"+ip+":2735", "v1.22", nil, nil)
	if err != nil {
		panic(err)
	}

	return  cli
}

func DeleteContainer(dockerClient *client.Client,num int)  {
	//timeout := 100 * time.Second
	fmt.Println("delete container: ","engine"+strconv.Itoa(num))
	/*err := dockerClient.ContainerStop(context.Background(),"engine"+strconv.Itoa(num),&timeout)
	if err!=nil{
		fmt.Println("stop error: ",err)
	}*/
	err1 := dockerClient.ContainerRemove(context.Background(),"engine"+strconv.Itoa(num),types.ContainerRemoveOptions{RemoveVolumes:true,Force:true})

	if err1!=nil{
		fmt.Println("delete error: ",err1)
	}
	waitGroup1.Done()

}


