package main

import (
	"fmt"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
)

func main() {
	//add ExecStart=/usr/bin/dockerd -H unix:///var/run/docker.sock -H tcp://ip:2735 in dokcer.service
	//systemctl daemon-reload & systemctl restart docker.service
	// and try "docker -H tcp://ip:2735 ps -a" to see if the setting works
	defaultHeaders := map[string]string{"User-Agent": "engine-api-cli-1.0"}
	cli, err := client.NewClient("tpc://ip:2735", "v1.22", nil, defaultHeaders)
	if err != nil {
		panic(err)
	}

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

