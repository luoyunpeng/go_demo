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

var (
	waitGroup sync.WaitGroup
 	ip string = "192.168.1.76"
	//dockerClient = InitClient(ip)
	containerHostConf = &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	imageName string = "api:4.0"
	name string = "engine"
	num int =20
	hostConf = make(map[string]string,num)
)

func main() {
	startTime := time.Now()
	fmt.Println("Creating containers....")
	waitGroup.Add(1)
	go PrepareContainer()
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds(),hostConf," seconds\n")

	//1 config hosts
	fmt.Println("Configuring the host....")
	waitGroup.Add(1)
	go ConfigHosts()
	fmt.Println("****wait for configure host file")
	waitGroup.Wait()

	//2 scp hosts
	waitGroup.Add(1)
	//go CopyHostToAll(dockerClient,"/opt/tmpconfig/hosts/scpHost.sh",hostConf)
	go ExecuteCMD([]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/scpHost.sh",name,strconv.Itoa(num)},true)
	fmt.Println("****wait for copying host configure file")
	waitGroup.Wait()

	//3 remove repeat hosts
	waitGroup.Add(1)
	//go RemoveRepeatHosts(dockerClient,"/opt/tmpconfig/hosts/removeRepeatHosts.sh",hostConf)
	go ExecuteCMD([]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/removeRepeatHosts.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for removing repeat host")
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds()," seconds\n")

	//4 configure zookeeper and HDFS, because this two component do not influence each other.
	//4.1.1 configure zookeeper
	fmt.Println("Configuring ans starting zookeeper & hdfs:")
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/zookeeper/zooConf.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for configuring  zookeeper....")

	//4.1.2 configure HDFS.
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/hdfs/hdfsConf.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for configuring  hdfs....")
	waitGroup.Wait()

	//4.2.1 scp zooConf
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/zookeeper/zooScp.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for scp  zoo.cfg....")

	//4.2.2 scp hdfs configure file
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/hdfs/hdfsScp.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for scp  hdfs configure file....")
	waitGroup.Wait()

	//4.3.1 start zookeeper
	waitGroup.Add(1)
	go ZookeeperStart()

	//4.3.2 start hdfs
	waitGroup.Add(1)
	go HDFSStart()
	waitGroup.Wait()

	//5 config hbase
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/hbase/hbaseConf.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for configuring  hbase....")
	waitGroup.Wait()

	//5.1
	waitGroup.Add(1)
	go ExecuteCMD([]string{name+strconv.Itoa(1), "/opt/tmpconfig/hbase/hbaseScp.sh", name, strconv.Itoa(num)},true)
	fmt.Println("****wait for scp  hbase configure file....")
	waitGroup.Wait()

	//5.2
	waitGroup.Add(1)
	go HBASEStart()
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds()," seconds\n")
}

func ConfigHosts()  {
	for containerName,ip := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD([]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/configHosts.sh",ip,containerName},true)
	}
	waitGroup.Done()
}

func PrepareContainer()  {
	//containerHostConf := &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	for i:=1;i<=num ;i++  {
		waitGroup.Add(1)
		go ContainStart(i,name)
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

func ExecuteCMD( cmd []string, isRoutineMode bool){
	dockerClient := InitClient(ip)
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
		_, errR := reader.ReadString('\n')
		if errR == io.EOF {
			break
		}
		//fmt.Println("message",readString)
	}

	if isRoutineMode {
		waitGroup.Done()
	}
}

func ContainStart( num int, name string) {
	dockerClient := InitClient(ip)
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
func Zookeeper(dockerClient *client.Client,cmd string)  {
	for containerName := range hostConf{
		waitGroup.Add(1)
		fmt.Println("copy to host: ",containerName)
		go ExecuteCMD([]string{name+strconv.Itoa(1),cmd,containerName,""},true)
	}
	waitGroup.Done()
}

//zookeeper
func ZookeeperStart()  {
	if num%2==0 {
		for i := 1;i<num ;i++  {
			waitGroup.Add(1)
			go ExecuteCMD([]string{name+strconv.Itoa(i), "/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"},true)
		}
	}else {
		for containerName := range hostConf{
			waitGroup.Add(1)
			go ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"},true)
		}
	}

	fmt.Println("****wait for starting zookeeper....")
	waitGroup.Done()
}

//hdfs start
func HDFSStart()  {
	//namenode format
	ExecuteCMD([]string{name+strconv.Itoa(1), "/usr/bin/hdfs", "namenode", "-format"},false)
	//start namenode
	ExecuteCMD([]string{name+strconv.Itoa(1), "/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start", "namenode"},false)

	//start datanode
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start","datanode"},true)
	}
	fmt.Println("****wait for starting hdfs....")
	waitGroup.Done()
}

func HBASEStart()  {
	//start master
	ExecuteCMD([]string{name+strconv.Itoa(1), "/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start", "master"},false)

	//start datanode
	for containerName := range hostConf{
		waitGroup.Add(1)
		go ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start","regionserver"},true)
	}
	fmt.Println("****wait for starting hbase....")
	waitGroup.Done()
}
