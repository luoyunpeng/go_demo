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
	"runtime"
)

var (
	waitGroup sync.WaitGroup
 	ip string = "192.168.1.76"
	containerHostConf = &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	imageName string = "api:4.0"
	name string = "engine"
	num int =8
	hostConf = make(map[string]string,num)
	allHDFS = make(chan  string)
)

func main() {
	runtime.GOMAXPROCS(4)

	startTime := time.Now()
	fmt.Println("Creating containers....")
	waitGroup.Add(1)
	go PrepareContainer()
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds(),"\n",hostConf," seconds\n")

	//1 config all container hosts
	go ConfigHosts()

	//2 configure and start zookeeper & HDFS, because this two component do not influence each other.
	go ZooConfAndStart()
	go HDFSConfAndStart()

	//3 hbase start
	waitGroup.Add(1)
	go HBASEConfAndStart()
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds()," seconds\n")
}

func ConfigHosts()  {
	fmt.Println("Configuring the host....")
	hostCh := make(chan string)

	fmt.Println("****wait for configure host file")
	configCh := make(chan  string)
	//1 configure hosts file
	go func(){
		for containerName,ip := range hostConf {
			ExecuteCMD([]string{name+strconv.Itoa(1),"/opt/tmpconfig/hosts/configHosts.sh",ip,containerName},false)
		}
		configCh <- "Done"
	}()

	fmt.Println("****wait for copying host configure file")
	scpCh := make(chan  string)
	//2 scp the hosts file to all container
	go func() {
		if <-configCh != "" {
			ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hosts/scpHost.sh", name, strconv.Itoa(num)}, false)
		}

		scpCh <- "Done"
	}()

	fmt.Println("****wait for removing repeat host")
	//3 remove repeat hosts configure
	go func() {
		if <-scpCh !="" {
			ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hosts/removeRepeatHosts.sh", name, strconv.Itoa(num)}, false)
		}
		hostCh <- "Done"
	}()

	<-hostCh
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

func ZooConfAndStart()  {
	fmt.Println("Configuring ans starting zookeeper & hdfs:")

	//1 config zookeeper
	fmt.Println("****wait for configuring  zookeeper....")
	zooConfCh := make(chan string)
	go func() {
		ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/zookeeper/zooConf.sh", name, strconv.Itoa(num)}, false)
		zooConfCh <- "Done"
	}()

	//2 scp zookeeper to container
	fmt.Println("****wait for scp  zoo.cfg....")
	zooScpCh := make(chan  string)
	go func() {
		if <-zooConfCh != "" {
			ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/zookeeper/zooScp.sh", name, strconv.Itoa(num)}, false)
		}
		zooScpCh <- "Done"
	}()

	//3 start zookeeper
	fmt.Println("****wait for starting zookeeper....")
	zooStartCh := make(chan  string)
	go func() {
		if <-zooScpCh != "" {
			if num%2 == 0 {
				for i := 1; i < num; i++ {
					waitGroup.Add(1)
					go ExecuteCMD([]string{name + strconv.Itoa(i), "/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"}, true)
				}
			} else {
				for containerName := range hostConf {
					waitGroup.Add(1)
					go ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"}, true)
				}
			}
		}
		zooStartCh <- "Done"
	}()

	<-zooStartCh
}

//hdfs start
func HDFSConfAndStart() {

	//1 config hdfs
	fmt.Println("****wait for configuring  hdfs....")
	hdfsConfCh := make(chan string)
	go func() {
		ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hdfs/hdfsConf.sh", name, strconv.Itoa(num)}, false)
		hdfsConfCh <- "Done"
	}()

	//2 scp hdfs configure file to all container
	fmt.Println("****wait for scp  hdfs configure file....")
	hdfsScpCh := make(chan string)
	go func() {
		if <-hdfsConfCh != "" {
			ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hdfs/hdfsScp.sh", name, strconv.Itoa(num)}, false)
		}
		hdfsScpCh <- "Done"
	}()

	//3 namenode format and start
	fmt.Println("****wait for formating and starting hdfs namenode....")
	namenodeCh := make(chan  string)
	go func() {
		if <-hdfsScpCh != "" {
			//namenode format
			ExecuteCMD([]string{name + strconv.Itoa(1), "/usr/bin/hdfs", "namenode", "-format"}, false)
			//start namenode
			ExecuteCMD([]string{name + strconv.Itoa(1), "/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start", "namenode"}, false)
		}
		namenodeCh <- "Done"
	}()

	//4 start datanode
	fmt.Println("****wait for starting hdfs datanode....")
	datanodeCh := make(chan  string)
	go func() {
		if <-namenodeCh != "" {
			for containerName := range hostConf {
				waitGroup.Add(1)
				go  ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start", "datanode"}, true)
			}
		}
		datanodeCh <- "Done"
	}()

	go func() {
		if <-datanodeCh != "" {
			allHDFS <- "Done"
		}
	}()
}

func ConfigHbase()  {
	fmt.Println("****wait for configuring  hbase....")
	//1 config hbase
	hbaseConfCh := make(chan string)
	go func() {
		ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hbase/hbaseConf.sh", name, strconv.Itoa(num)}, false)
		hbaseConfCh <- "Done"
	}()

	//2 scp hbase configure file to all container
	fmt.Println("****wait for scp  hbase configure file....")
	hbaseScpCh := make(chan string)
	go func() {
		if <-hbaseConfCh != "" {
			ExecuteCMD([]string{name + strconv.Itoa(1), "/opt/tmpconfig/hbase/hbaseScp.sh", name, strconv.Itoa(num)}, false)
		}
		hbaseScpCh <- "Done"
	}()

	<-hbaseScpCh
}

func HBASEConfAndStart()  {
	hbaseConfCh := make(chan string)
	//1 configure hbase
	go func() {
		ConfigHbase()
		hbaseConfCh <- "Done"
	}()

	hbaseMasterCh := make(chan string)
	//2  start master
	go func() {
		if <-allHDFS !="" && <-hbaseConfCh !="" {
			fmt.Println("****wait for starting hbase....")
			ExecuteCMD([]string{name + strconv.Itoa(1), "/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start", "master"}, false)
		}
		hbaseMasterCh <- "Done"
	}()

	//3 start all datanode
	hbaseRStartCh := make(chan string)
	go func() {
		if <-hbaseMasterCh !="" {
			for containerName := range hostConf {
				waitGroup.Add(1)
				go ExecuteCMD([]string{containerName, "/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start", "regionserver"}, true)
			}
		}
		hbaseRStartCh <- "Done"
	}()
	<-hbaseRStartCh
	waitGroup.Done()
}
