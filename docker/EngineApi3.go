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
	"io"
)

var waitGroup sync.WaitGroup

const  (
	imageName string = "api:2.0"
	name string = "engine"
	num int =30
)

func main() {
	startTime := time.Now()
	hostConf := make(map[string]string,num)
	dockerClient := InitClient("ip")

	fmt.Println("create container ")
	waitGroup.Add(1)
	go PrepareContainer(dockerClient,hostConf)
	waitGroup.Wait()

	fmt.Println("it takes ",time.Now().Sub(startTime)," to create containers",hostConf)

	fmt.Println("create done, starting to configure ths hosts file")
	//1 config hosts
	waitGroup.Add(1)
	go ConfigHosts(dockerClient,"/opt/tmpconfig/hosts/configHosts.sh",hostConf)
	fmt.Println("wait for configure host")
	waitGroup.Wait()

	//2 scp hosts
	waitGroup.Add(1)
	//go CopyHostToAll(dockerClient,"/opt/tmpconfig/hosts/scpHost.sh",hostConf)
	go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/scpHost.sh",name,strconv.Itoa(num)})
	fmt.Println("wait for copying host")
	waitGroup.Wait()

	//3 remove repeat hosts
	waitGroup.Add(1)
	//go RemoveRepeatHosts(dockerClient,"/opt/tmpconfig/hosts/removeRepeatHosts.sh",hostConf)
	go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/removeRepeatHosts.sh",name,strconv.Itoa(num)})
	fmt.Println("wait for removing repeat host")
	waitGroup.Wait()
	fmt.Println("it takes ",time.Now().Sub(startTime)," to create and config containers")

	//4 configure zookeeper
	fmt.Println("starting to configure zookeeper")
	waitGroup.Add(1)
	go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),"/opt/tmpconfig/zookeeper/zooConf.sh",name,strconv.Itoa(num)})
	fmt.Println("wait for configure zookeeper")
	waitGroup.Wait()

	//5 scp zooConf
	waitGroup.Add(1)
	//go  Zookeeper(dockerClient,"/opt/tmpconfig/zookeeper/zooScp.sh",hostConf)
	go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),"/opt/tmpconfig/zookeeper/zooScp.sh",name,strconv.Itoa(num)})
	fmt.Println("wait for scp  zoo.cfg")
	waitGroup.Wait()
	time.Sleep(10*1e9)

	//6 start zookeeper
	fmt.Println("start zookeeper")
	waitGroup.Add(1)
	go ZookeeperStart(dockerClient,"/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh",hostConf)
	fmt.Println("wait for starting zookeeper")
	waitGroup.Wait()

	fmt.Println("it takes ",time.Now().Sub(startTime)," to start zookeeper")
}
/*
func RemoveRepeatHosts(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,[]string{containerName,cmd,"",""})
	}
	waitGroup.Done()
}

func CopyHostToAll(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),cmd,containerName,""})
	}
	waitGroup.Done()
}*/

func ConfigHosts(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName,ip := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),cmd,ip,containerName})
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

func ExecuteCMD(dockerClient *client.Client,  cmd []string){
	execResponse,err := dockerClient.ContainerExecCreate(context.Background(), cmd[0] ,types.ExecConfig{User: "root",Cmd: cmd[1:],Privileged: true,Tty:true,AttachStdout:true})

	if err!=nil {
		panic(err)
	}

	//dockerClient.ContainerExecStart(context.Background(),execResponse.ID,types.ExecStartCheck{Detach: true, Tty:false})
	response,err := dockerClient.ContainerExecAttach(context.Background(),execResponse.ID,types.ExecConfig{AttachStdin:true,AttachStdout:true,User: "root"})
	if err!=nil {
		panic(err)
	}
	reader := response.Reader
	for   {
		readString, errR := reader.ReadString('\n')
		if errR == io.EOF {
			break
		}
		fmt.Println("message",readString)
	}

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
		//panic(err)
		fmt.Println("get error: ",err)
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

//zookeeper
func Zookeeper(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		fmt.Println("copy to host: ",containerName)
		go ExecuteCMD(dockerClient,[]string{name+strconv.Itoa(1),cmd,containerName,""})
	}
	waitGroup.Done()
}

//zookeeper
func ZookeeperStart(dockerClient *client.Client,cmd string,hostConf map[string]string)  {
	if num%2==0 {
		for i:= 1;i<num ;i++  {
			waitGroup.Add(1)
			go ExecuteCMD(dockerClient, []string{name+strconv.Itoa(i), cmd, "start", ""})
		}
	}else {
		for containerName := range hostConf{
			waitGroup.Add(1)
			go ExecuteCMD(dockerClient, []string{containerName, cmd, "start", ""})
		}
	}

	waitGroup.Done()
}

