package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func GetEndpointSettings(cli *client.Client, containerId string) *network.EndpointSettings {
	container, err := cli.ContainerInspect(context.Background(), containerId)
	if err != nil {
		log.Fatal(err)
	}
	endpoint := container.NetworkSettings.Networks["devnet"]
	return endpoint.Copy()
}
