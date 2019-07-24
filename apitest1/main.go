package main

import (
	"io"
	"os"
	golog "log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/filters"
	"time"
)

func main() {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(ctx)

	desiredimage:=			"ubuntu:latest"
	desirednetwork := 		"testnet1"
	desiredcontainername := "testhost1"

	// create a network
	networkID, err := createNetwork(cli, desirednetwork)
	if err!=nil{
		golog.Fatalln(err.Error())
	}
	golog.Println("created network with id: ", networkID)


	// search for image in local images
	imageListResults, err := cli.ImageList(ctx, types.ImageListOptions{
		All:     false,
		Filters: filters.Args{},
	})
	if err!=nil{
		golog.Println("error searching locally for images: ", err.Error())
		reader, err := cli.ImagePull(ctx, "ubuntu:latest", types.ImagePullOptions{})
		if err != nil {
			panic(err)
		}
		io.Copy(os.Stdout, reader)
	}
	golog.Println("Found matching images:")
	imagename := ""
	for _,image := range imageListResults {
		for _,tag := range image.RepoTags {
			if tag==desiredimage {
				imagename = tag
			}
		}
		if imagename!="" {
			break
		}
	}

	// create container
	containerID, err := createContainer(cli, desiredcontainername, imagename)
	if err!=nil{
		golog.Fatalln(err.Error())
	}
	golog.Println("Created container ID: ", containerID)

	// attach the container to the network
	ctx1,_ := context.WithTimeout(ctx, time.Duration(5 * time.Second))
	if err :=cli.NetworkConnect(ctx1, networkID, containerID, nil); err!=nil{
		golog.Fatalln("error attaching container to the network: ", err.Error())
	}
	golog.Println("attached container to network")

	// start and wait on the container
	if err := cli.ContainerStart(ctx, containerID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	golog.Println("container started ...")



	statusCh, errCh := cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
		case err := <-errCh:
			if err != nil {
				panic(err)
			}
		case <-statusCh:

			case <-time.After(2 * time.Second):
				out, err := cli.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{ShowStdout: true})
				if err != nil {
					panic(err)
				}
				io.Copy(os.Stdout, out)
	}



}

func createContainer(cli *client.Client, containername string, imagename string) (string, error) {

	ctx := context.Background()

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: imagename,
		Cmd:   []string{"/bin/bash"},
		Tty:   true,
	}, nil, nil, containername)
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func createNetwork(cli *client.Client, netname string) (string,error){

	ctx := context.Background()

	golog.Println("Create network ...")
	networkconfig := network.IPAMConfig{
		Subnet:     "10.10.10.0/24",
		IPRange:    "",
		Gateway:    "",
		AuxAddress: nil,
	}
	networkresp, err := cli.NetworkCreate(ctx, netname, types.NetworkCreate{
		CheckDuplicate: true,
		Driver:         "bridge",
		EnableIPv6:     false,
		IPAM:           &network.IPAM{
			Driver:  "default",
			Options: nil,
			Config:  []network.IPAMConfig{networkconfig},
		},
		Internal:       true,
		Attachable:     true,
		Ingress:        false,
		ConfigOnly:     false,
		ConfigFrom:     nil,
		Options:        nil,
		Labels:         nil,
	})
	if err!=nil{
		return "", err
	}

	return networkresp.ID, nil


}
