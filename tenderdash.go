package main

import (
	"context"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func SetupTenderdash(cli *client.Client, mnode *MasternodeConfig, chainlock Chainlock) <-chan error {
	resChannel := make(chan error)
	go func() {
		volumeName := mnode.Name + "_drive_tenderdash_data"
		containerName := mnode.Name + "_drive_tenderdash"

		vol, err := cli.VolumeCreate(context.Background(), volume.VolumeCreateBody{
			Name: volumeName,
		})
		if err != nil {
			log.Fatal(err)
		}
		mnode.TenderdashVolume = &vol

		// Initialize tenderdash
		body, err := cli.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: "dashpay/tenderdash:0.6.0",
				Cmd:   strslice.StrSlice{"init"},
			},
			&container.HostConfig{
				RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
				Binds: []string{
					volumeName + ":/tenderdash",
				},
			},
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					"devnet": {},
				},
			},
			&v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			},
			containerName,
		)
		log.Printf("Initialized tenderdash for %s\n", mnode.Name)

		time.Sleep(5 * time.Second)
		cli.ContainerStop(context.Background(), body.ID, nil)
		cli.ContainerRemove(context.Background(), body.ID, types.ContainerRemoveOptions{})

		body, err = cli.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: "dashpay/tenderdash:0.6.0",
			},
			&container.HostConfig{
				RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
				Binds: []string{
					volumeName + ":/tenderdash",
				},
			},
			&network.NetworkingConfig{
				EndpointsConfig: map[string]*network.EndpointSettings{
					"devnet": {},
				},
			},
			&v1.Platform{
				Architecture: "amd64",
				OS:           "linux",
			},
			containerName,
		)

		cli.ContainerExecCreate(context.Background(), body.ID, types.ExecConfig{Cmd: []string{}})

		log.Printf("Started tenderdash for %s\n", mnode.Name)

		mnode.TenderdashContainerId = body.ID
	}()
	return resChannel
}

func StopTenderdash(cli *client.Client, mnode *MasternodeConfig) {
	cli.ContainerStop(context.Background(), mnode.DriveContainerId, nil)
}
