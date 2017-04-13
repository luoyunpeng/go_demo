package main


import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"github.com/docker/engine-api/types/container"
	"strconv"
)

var ContainerMap map[int]string

func main() {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tcp://ip:2735", "v1.22", nil, defaultHeaders)
	if err != nil {
		panic(err)
	}
	//ContainStart(cli,1,"goapi")

	TestExec(cli,"goapi1")
}

func TestExec(cli *client.Client,  name string){
	                                                                                                        // for cmd param, the first start is the main command , and another is param
														// we can set like "[]string{"/test.sh","name"}" ,means give param "name" to shell test.sh
	responseExc,err := cli.ContainerExecCreate(context.Background(),name,types.ExecConfig{User: "root",Cmd: []string{"/test.sh"},Privileged: true})

	if err!=nil {
		fmt.Println(err)
	}
	cli.ContainerExecStart(context.Background(),responseExc.ID,types.ExecStartCheck{Detach: true, Tty:false})


}

func ContainStart(cli *client.Client, num int, name string)  {
	resp, err := cli.ContainerCreate(context.Background(), &container.Config{
		Image: 		"newhdp:2.4",
		Cmd:   		[]string{"/service.sh"},
		Hostname: 	name+ strconv.Itoa(num),
	}, &container.HostConfig{
		Privileged: true,
		PublishAllPorts: true,
		NetworkMode:  "hadoop",
	}, nil, name+ strconv.Itoa(num))
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	networkResource,err := cli.NetworkInspect(context.Background(),"hadoop")
	if err != nil {
		panic(err)
	}
	fmt.Println("the container ID: ",resp.ID,networkResource.Containers[resp.ID].IPv4Address,networkResource.Containers[resp.ID].Name)

}

func GetContainList()  {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tpc://ip:2735", "v1.22", nil, defaultHeaders)
	if err != nil {
		panic(err)
	}
	fmt.Println("client message: ")
	options := types.ContainerListOptions{All: true}

	containers, err := cli.ContainerList(context.Background(), options)
	if err != nil {
		panic(err)
	}

	fmt.Println(cli.ClientVersion())

	for _, c := range containers {
		fmt.Println(c.ID, c.Command, c.Names, c.SizeRootFs, c.ImageID, c.NetworkSettings)
	}

	//cli.ContainerStop(context.Background(), "test", nil)
}

