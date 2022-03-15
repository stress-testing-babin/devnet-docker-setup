package main

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func SetupDrive(cli *client.Client, mnode *MasternodeConfig, chainlock Chainlock) <-chan error {
	resChannel := make(chan error)
	go func() {
		volumeName := mnode.Name + "_drive_abci_data"
		containerName := mnode.Name + "_drive_abci"
		log.Println("Setting up " + containerName)

		vol, err := cli.VolumeCreate(context.Background(), volume.VolumeCreateBody{
			Name: volumeName,
		})
		if err != nil {
			log.Fatal(err)
		}
		mnode.DriveVolume = &vol

		body, err := cli.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: "dashpay/drive:0.21.1",
				Cmd: strslice.StrSlice{
					"npm",
					"run",
					"abci",
				},
				Env: []string{
					"ABCI_PORT=26658",
					"CORE_JSON_RPC_USERNAME=dashrpc",
					"CORE_JSON_RPC_PASSWORD=rpcpassword",
					"CORE_JSON_RPC_HOST=" + mnode.Target.Host,
					"CORE_JSON_RPC_PORT=" + RPC_PORT,
					"CORE_ZMQ_HOST=" + mnode.Target.Host,
					"CORE_ZMQ_PORT=29998",
					"DOCUMENT_MONGODB_URL=mongodb://drive_mongodb:27017?replicaSet=driveDocumentIndices",
					"NODE_ENV=development",
					"LOG_STDOUT_LEVEL=debug",
					"LOG_PRETTY_FILE_LEVEL=debug",
					"LOG_PRETTY_FILE_PATH=/var/log/drive-pretty.log",
					"LOG_JSON_FILE_LEVEL=info",
					"LOG_JSON_FILE_PATH=/var/log/drive-json.log",
					"INITIAL_CORE_CHAINLOCKED_HEIGHT=" + fmt.Sprint(chainlock.Height),
				},
			},
			&container.HostConfig{
				RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
				Binds: []string{
					volumeName + ":/usr/src/app/db",
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
		mnode.DriveContainerId = body.ID
		log.Println(containerName + " created")

		err = cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{})
		log.Println(containerName + " started")
	}()
	return resChannel
}

func StopDrive(cli *client.Client, mnode *MasternodeConfig) {
	cli.ContainerStop(context.Background(), mnode.DriveContainerId, nil)
}
