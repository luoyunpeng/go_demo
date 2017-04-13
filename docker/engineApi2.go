package main


import (
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"github.com/docker/engine-api/types/container"
	"strconv"
	"strings"
	"sync"
	"time"
)

var waitGroup sync.WaitGroup
const  (
	imageName string = "api:1.0"
	name string = "engine"
	num int =10
)
func main() {
	startTime := time.Now()
	hostConf := make(map[string]string,num)
	dockerClient := InitClient("192.168.1.76")

	fmt.Println("create container ")
	waitGroup.Add(1)
	go PrepareContainer(dockerClient,hostConf)
	waitGroup.Wait()

	fmt.Println("it takes ",time.Now().Sub(startTime)," to create containers",hostConf)

	fmt.Println("create done, starting to configure")
	//1 config hosts
	waitGroup.Add(1)
	go ConfigHosts(dockerClient,"/opt/tmpconfig/hosts/configHosts.sh",hostConf)
	waitGroup.Wait()

	//2 scp hosts
	waitGroup.Add(1)
	go CopyHostToAll(dockerClient,"/opt/tmpconfig/hosts/scpHost.sh",hostConf)
	waitGroup.Wait()

	//3 remove repeat hosts
	waitGroup.Add(1)
	go RemoveRepeatHosts(dockerClient,"/opt/tmpconfig/hosts/removeRepeatHosts.sh ",hostConf)
	waitGroup.Wait()
	fmt.Println("it takes ",time.Now().Sub(startTime)," to create and config containers")
}

func RemoveRepeatHosts(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,containerName,cmd,containerName,"")
	}
	waitGroup.Done()
}

func CopyHostToAll(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,name+strconv.Itoa(1),cmd,containerName,"")
	}
	waitGroup.Done()
}

func ConfigHosts(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName,ip := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,name+strconv.Itoa(1),cmd,ip,containerName)
	}
	waitGroup.Done()
}

func PrepareContainer(dockerClient *client.Client,hostConf map[string]string )  {
	containerHostConf := &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	for i:=1;i<=num ;i++  {
		waitGroup.Add(1)
		go ContainStart(dockerClient,i,name,containerHostConf,hostConf)
	}
	waitGroup.Done()
}

func InitClient(ip string) *client.Client  {
	//defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tcp://"+ip+":2735", "v1.22", nil, nil)
	if err != nil {
		panic(err)
	}

	return  cli
}

func ExecuteCMD(dockerClient *client.Client,  executeCName,cmd,firstParam,secondParam string){
	fmt.Println("execute the command ",cmd+firstParam+secondParam)
	execResponse,err := dockerClient.ContainerExecCreate(context.Background(), executeCName ,types.ExecConfig{User: "root",Cmd: []string{cmd,firstParam,secondParam },Privileged: true})

	if err!=nil {
		panic(err)
	}

	dockerClient.ContainerExecStart(context.Background(),execResponse.ID,types.ExecStartCheck{Detach: true, Tty:false})
	waitGroup.Done()
}

func ContainStart(dockerClient *client.Client, num int, name string, containerHostConf  *container.HostConfig,hostConf map[string]string) {
	dockerContainer, err := dockerClient.ContainerCreate(context.Background(), & container.Config{
		Image: imageName,
		Cmd: []string{"/service.sh"},
		Hostname: name+ strconv.Itoa(num),
	}, containerHostConf, nil, name+ strconv.Itoa(num))
	if err != nil {
		panic(err)
	}

	if err := dockerClient.ContainerStart(context.Background(), dockerContainer.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	networkResource,err := dockerClient.NetworkInspect(context.Background(),"hadoop")
	if err != nil {
		panic(err)
	}

	hostConf[networkResource.Containers[dockerContainer.ID].Name] = strings.Split(networkResource.Containers[dockerContainer.ID].IPv4Address,"/")[0]
	//fmt.Println("the container ID: ",dockerContainer.ID,networkResource.Containers[dockerContainer.ID].IPv4Address,networkResource.Containers[dockerContainer.ID].Name)
	waitGroup.Done()
}

func GetContainList()  {
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tpc://192.168.1.76:2735", "v1.22", nil, defaultHeaders)
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

