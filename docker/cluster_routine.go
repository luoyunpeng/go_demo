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
	"os"
	"flag"
)

type ClusterConf struct {
	ImageName string
	Name string
	Size int
	ProcNum int
}

var (
	waitGroup sync.WaitGroup
 	ip string = "192.168.1.76"
	containerHostConf = &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	hostConf map[string]string
	allHDFS = make(chan  string)
	Conf *ClusterConf
)

func main()  {

	if len(os.Args) <=1 {
		fmt.Println("Please input parameter\n")
		PrintUsage()
		return
	}

	commands := strings.Join(os.Args[1:]," ")
	if strings.Contains(commands,"-h") ||strings.Contains(commands,"help") ||strings.Contains(commands,"--help")    {
		PrintUsage()
		return
	}
	parseParam()
	ClusterConfAndStart()
}

func ClusterConfAndStart() {
	runtime.GOMAXPROCS(Conf.ProcNum)

	startTime := time.Now()
	fmt.Println("Creating containers....")
	waitGroup.Add(1)
	go PrepareContainer()
	waitGroup.Wait()
	fmt.Println("Done, takes ",time.Now().Sub(startTime).Seconds()," seconds\n",hostConf)

	//1 config all container hosts file , and config and start zookeeper & hdfs, and Hbase in concurrent way.
	//because of these steps don't influence each other start and after these all done, start Hbase
	if Conf.Size>1 {
		go ConfigHosts()
	}
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
			ExecuteCMD(Conf.Name+strconv.Itoa(1),[]string{"/opt/tmpconfig/hosts/configHosts.sh",ip,containerName},false)
		}
		configCh <- "Done"
	}()

	fmt.Println("****wait for copying host configure file")
	scpCh := make(chan  string)
	//2 scp the hosts file to all container
	go func() {
		if <-configCh != "" {
			ExecuteCMD(Conf.Name+strconv.Itoa(1),[]string{ "/opt/tmpconfig/hosts/scpHost.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		}

		scpCh <- "Done"
	}()

	fmt.Println("****wait for removing repeat host")
	//3 remove repeat hosts configure
	go func() {
		if <-scpCh !="" {
			ExecuteCMD(Conf.Name+strconv.Itoa(1),[]string{ "/opt/tmpconfig/hosts/removeRepeatHosts.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		}
		hostCh <- "Done"
	}()

	<-hostCh
}

func PrepareContainer()  {
	//containerHostConf := &container.HostConfig{ Privileged: true, PublishAllPorts: true, NetworkMode:  "hadoop" }
	for i:=1;i<=Conf.Size ;i++  {
		waitGroup.Add(1)
		go ContainStart(i,Conf.Name)
	}
	waitGroup.Done()
}

func InitClient(ip string) *client.Client  {
	//defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tcp://"+ip+":2735", "v1.22", nil, nil)
	//cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return  cli
}

func ExecuteCMD( execContainerName string,cmd []string, isRoutineMode bool){
	dockerClient := InitClient(ip)
	execResponse,err := dockerClient.ContainerExecCreate(context.Background(), execContainerName,types.ExecConfig{User: "root",Cmd: cmd,Privileged: true,Tty:true,AttachStdout:true})

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
		//fmt.Println("message: ",readString)
	}

	if isRoutineMode {
		waitGroup.Done()
	}
}

func ContainStart( num int, name string) {
	dockerClient := InitClient(ip)
	dockerContainer, err := dockerClient.ContainerCreate(context.Background(), & container.Config{
		Image: Conf.ImageName,
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

func isNameExits(name string) bool {
	dockerClient := InitClient(ip)
	options := types.ContainerListOptions{All: true}
	containers, err := dockerClient.ContainerList(context.Background(), options)
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		if strings.Contains(strings.Join(container.Names,""), name) {
			return true
		}
	}

	return false
}

func ZooConfAndStart()  {
	fmt.Println("Configuring ans starting zookeeper & hdfs:")

	//1 config zookeeper
	fmt.Println("****wait for configuring  zookeeper....")
	zooConfCh := make(chan string)
	go func() {
		ExecuteCMD(Conf.Name+strconv.Itoa(1),[]string{"/opt/tmpconfig/zookeeper/zooConf.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		zooConfCh <- "Done"
	}()

	//2 scp zookeeper to container
	fmt.Println("****wait for scp  zoo.cfg....")
	zooScpCh := make(chan  string)
	go func() {
		if <-zooConfCh != "" {
			ExecuteCMD(Conf.Name+strconv.Itoa(1),[]string{"/opt/tmpconfig/zookeeper/zooScp.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		}
		zooScpCh <- "Done"
	}()

	//3 start zookeeper
	fmt.Println("****wait for starting zookeeper....")
	zooStartCh := make(chan  string)
	go func() {
		if <-zooScpCh != "" {
			if Conf.Size%2 == 0 {
				for i := 1; i < Conf.Size; i++ {
					waitGroup.Add(1)
					go ExecuteCMD(Conf.Name + strconv.Itoa(i),[]string{"/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"}, true)
				}
			} else {
				for containerName := range hostConf {
					waitGroup.Add(1)
					go ExecuteCMD(containerName, []string{"/usr/hdp/2.4.0.0-169/zookeeper/bin/zkServer.sh", "start"}, true)
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
		ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/opt/tmpconfig/hdfs/hdfsConf.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		hdfsConfCh <- "Done"
	}()

	//2 scp hdfs configure file to all container
	fmt.Println("****wait for scp  hdfs configure file....")
	hdfsScpCh := make(chan string)
	go func() {
		if <-hdfsConfCh != "" {
			ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/opt/tmpconfig/hdfs/hdfsScp.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		}
		hdfsScpCh <- "Done"
	}()

	//3 namenode format and start
	fmt.Println("****wait for formating and starting hdfs namenode....")
	namenodeCh := make(chan  string)
	go func() {
		if <-hdfsScpCh != "" {
			//namenode format
			ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/usr/bin/hdfs", "namenode", "-format"}, false)
			//start namenode
			ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start", "namenode"}, false)
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
				go  ExecuteCMD(containerName, []string{"/usr/hdp/2.4.0.0-169/hadoop/sbin/hadoop-daemon.sh", "start", "datanode"}, true)
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
		ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/opt/tmpconfig/hbase/hbaseConf.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
		hbaseConfCh <- "Done"
	}()

	//2 scp hbase configure file to all container
	fmt.Println("****wait for scp  hbase configure file....")
	hbaseScpCh := make(chan string)
	go func() {
		if <-hbaseConfCh != "" {
			ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/opt/tmpconfig/hbase/hbaseScp.sh", Conf.Name, strconv.Itoa(Conf.Size)}, false)
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
			ExecuteCMD(Conf.Name + strconv.Itoa(1), []string{"/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start", "master"}, false)
		}
		hbaseMasterCh <- "Done"
	}()

	//3 start all datanode
	hbaseRStartCh := make(chan string)
	go func() {
		if <-hbaseMasterCh !="" {
			for containerName := range hostConf {
				waitGroup.Add(1)
				go ExecuteCMD(containerName, []string{"/usr/hdp/2.4.0.0-169/hbase/bin/hbase-daemon.sh", "start", "regionserver"}, true)
			}
		}
		hbaseRStartCh <- "Done"
	}()
	<-hbaseRStartCh
	waitGroup.Done()
}

func parseParam()  {
	imageName := flag.String("image","","base image use to create container")
	procNum := flag.Int("procNum",1,"processor number used by go routine")
	size := flag.Int("size",0,"container size")
	name := flag.String("name","","the prefix name of container")

	flag.Parse()

	if *imageName =="" {
		fmt.Println("imageName must provide")
		return
	}

	if *name =="" {
		fmt.Println("the prefix name must provide")
		return
	}else {
		if isNameExits(*name) {
			fmt.Println(*name," is already the prefix name of running container. please check and input a unique prefix name")
			os.Exit(1)
		}
	}

	if *size <= 0 {
		fmt.Println("the cluster size must bigger than 0")
		return
	}else if *size >30 {
		fmt.Println("the cluster size must less than 40")
		return
	}

	if *size < 0 {
		fmt.Println("the cluster size must bigger than 0")
		return
	}

	if *procNum > runtime.NumCPU() {
		fmt.Println("Warning: the procNum bigger than current processor number, set to ",runtime.NumCPU())
		*procNum = runtime.NumCPU()
	}else if *procNum < 0 {
		fmt.Println("Warning: the procNum less than 0, set to default 1")
		*procNum = 1
	}

	Conf = &ClusterConf{*imageName,*name,*size,*procNum}
	hostConf = make(map[string]string,*size)
}

func PrintUsage() {
	fmt.Println("Usage: cluster [OPTIONS] \n")
	fmt.Println("Options:")
	fmt.Println("--image","             The docker image that use to build hadoop cluster")
	fmt.Println("--procNum","           The number of processor that go routine will use")
	fmt.Println("--size","              The number of container")
	fmt.Println("--name","              The prefix name of container")
}
