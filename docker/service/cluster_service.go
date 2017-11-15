package main

import (
	"context"
	"fmt"
	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/mount"
	"github.com/docker/engine-api/types/swarm"
	"strconv"
	"sync"
)

var (
	waitGroup_service        = sync.WaitGroup{}
	dockerServer             = "192.168.1.165"
	servicePrefixName        = "demo"
	replicas          uint64 = 1
	serviceNum = 3
	networkName              = "zookeeper"
	targetPort        uint32 = 6666
	ZOO_SERVERS              = ""
	//service mode
	mode = swarm.ServiceMode{Replicated: &swarm.ReplicatedService{Replicas: &replicas}}
	//service network
	network = []swarm.NetworkAttachmentConfig{swarm.NetworkAttachmentConfig{Target: networkName}}
)

func main() {
	for i := 1; i <= serviceNum; i++ {
		ZOO_SERVERS += "server." + strconv.Itoa(i) + "=" + servicePrefixName + strconv.Itoa(i) + ":2888:3888  "
	}

	for i := 1; i <= serviceNum; i++ {
		waitGroup_service.Add(1)
		fmt.Println("start to create service "+servicePrefixName, i, "....")
		go CreateService(servicePrefixName+strconv.Itoa(i), targetPort, targetPort+(uint32(i-1)), i, ZOO_SERVERS)
	}

	waitGroup_service.Wait()
	fmt.Println("done")
}

func CreateService(serviceName string, targetPort uint32, publishedPort uint32, id int, servers string) {
	dockerClient := initClient(dockerServer)

	//service name and lables
	anno := swarm.Annotations{Name: serviceName}

	//port
	endpoint := &swarm.EndpointSpec{
		Ports: []swarm.PortConfig{
			swarm.PortConfig{
				TargetPort:    targetPort,
				PublishedPort: publishedPort,
			},
		},
	}

	//container template
	//1.1 volume
	dataMount := mount.Mount{
		Type:   mount.TypeVolume,
		Source: serviceName + "_data",
		Target: "/data",
	}

	logMount := mount.Mount{
		Type:   mount.TypeVolume,
		Source: serviceName + "_log",
		Target: "/datalog",
	}

	//1.2 container
	container := swarm.ContainerSpec{
		Image:  "zookeeper",
		Env:    []string{"ZOO_MY_ID=" + strconv.Itoa(id), "ZOO_SERVERS=" + servers},
		Mounts: []mount.Mount{dataMount, logMount},
	}

	// task
	task := swarm.TaskSpec{
		ContainerSpec: container,
		Networks:      network,
	}

	service := swarm.ServiceSpec{
		Annotations:  anno,
		TaskTemplate: task,
		Mode:         mode,
		EndpointSpec: endpoint,
	}

	serviceId,err := dockerClient.ServiceCreate(context.Background(), service, types.ServiceCreateOptions{})
	if err!=nil {
		fmt.Println(err)
	}
	
	fmt.Println("service Id",serviceId)
	waitGroup_service.Done()
}

func initClient(ip string) *client.Client {
	//defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	//cli, err := client.NewClient("tcp://"+dockerServer+":2376", "v1.32", nil, nil)
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	fmt.Println(client.DefaultDockerHost)
	return cli
}
