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

func SetupMongo(cli *client.Client, mnode *MasternodeConfig) <-chan error {
	resChannel := make(chan error)
	go func() {
		volumeName := mnode.Name + "_drive_mongodb_data"
		containerName := mnode.Name + "_drive_mongodb"
		log.Println("Setting up " + containerName)

		vol, err := cli.VolumeCreate(context.Background(), volume.VolumeCreateBody{
			Name: volumeName,
		})
		if err != nil {
			log.Fatal(err)
		}
		mnode.MongoVolume = &vol

		body, err := cli.ContainerCreate(
			context.Background(),
			&container.Config{
				Image: "mongo:4.2",
				Cmd: strslice.StrSlice{
					"mongod",
					"--replSet",
					"driveDocumentIndices",
					"--bind_ip_all",
				},
			},
			&container.HostConfig{
				Binds: []string{
					volumeName + ":/data/db",
					"/home/pit/code/dash_stress_testing/docker-compose/mongodb/initiate_mongodb_replica.js:/docker-entrypoint-initdb.d/initiate_mongodb_replica.js",
				},
				ExtraHosts:    []string{containerName + ":127.0.0.1"},
				RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
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
		if err != nil {
			log.Fatal(err)
		}
		log.Println(containerName + " created")
		mnode.MongoContainerId = body.ID

		err = cli.ContainerStart(context.Background(), body.ID, types.ContainerStartOptions{})
		log.Println(containerName + " started")
		if err != nil {
			log.Fatal(err)
		}

		resChannel <- nil
	}()
	return resChannel
}

func StopMongo(cli *client.Client, mnode *MasternodeConfig) {
	cli.ContainerStop(context.Background(), mnode.MongoContainerId, nil)
}
