package main

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// Create a new fullnode.
// Returns the hostname and the container id.
func CreateFullnode(cli *client.Client, name string, cmd strslice.StrSlice, nodeVolume *types.Volume, givenEndpointConf *network.EndpointSettings) (string, string, *types.Volume, *network.EndpointSettings, error) {
	volumeName := name + "_core_data"
	containerName := name + "_core"

	if nodeVolume == nil { // If no volume was given, generate a new one
		vol, err := cli.VolumeCreate(context.Background(), volume.VolumeCreateBody{
			Name: volumeName,
		})
		if err != nil {
			return "Unknown", "", nil, nil, err
		}
		nodeVolume = &vol
	}

	endpointSettings := &network.EndpointSettings{}

	if givenEndpointConf != nil {
		endpointSettings = givenEndpointConf.Copy()
		log.Println("Copying cointainer network config")
	}

	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			"devnet": endpointSettings,
		},
	}

	body, err := cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: "dashpay/dashd:0.17",
			Cmd:   cmd,
		},
		&container.HostConfig{
			Binds: []string{
				volumeName + ":/dash",
				"/home/pit/code/dash_stress_testing/docker-compose/core/dash.conf:/dash/.dashcore/dash.conf",
				//"./wallets/${WALLET:?err}:/dash/${WALLET:?err}",
			},
		},
		networkConfig,
		&v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
		containerName,
	)
	if err != nil {
		return "Unknown", "", nil, nil, err
	}

	err = cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{})
	if err != nil {
		return "Unknown", "", nil, nil, err
	}

	return containerName, body.ID, nodeVolume, endpointSettings, nil
}
