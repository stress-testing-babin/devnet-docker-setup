package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
)

const DEVNET_PARAMS = "3:2"

type Node struct {
	Host string
	Id   string
}

var dockerClient *client.Client

func InitDockerClient() {
	var err error
	dockerClient, err = client.NewEnvClient()
	if err != nil {
		fmt.Println("Unable to create docker client")
		log.Fatal(err)
	}
}

func CreateSeed() (Node, error) {
	seedNodeHost, seedNodeId, _, _, err := CreateFullnode(
		dockerClient,
		"go_seed",
		strslice.StrSlice{
			"dashd",
			"-llmqdevnetparams=" + DEVNET_PARAMS,
			"-llmqchainlocks=llmq_devnet",
			"-llmqinstantsend=llmq_devnet",
		},
		nil,
		nil,
	)
	if err != nil {
		return Node{}, err
	}

	return Node{
		Host: seedNodeHost,
		Id:   seedNodeId,
	}, nil
}

func ActivateSporks(target *RpcTarget) error {
	err := Spork(target, "SPORK_2_INSTANTSEND_ENABLED", 0)
	if err != nil {
		return err
	}
	err = Spork(target, "SPORK_3_INSTANTSEND_BLOCK_FILTERING", 0)
	if err != nil {
		return err
	}
	err = Spork(target, "SPORK_9_SUPERBLOCKS_ENABLED", 0)
	if err != nil {
		return err
	}
	err = Spork(target, "SPORK_17_QUORUM_DKG_ENABLED", 0)
	if err != nil {
		return err
	}
	err = Spork(target, "SPORK_19_CHAINLOCKS_ENABLED", 0)
	if err != nil {
		return err
	}
	//Spork(targetSeedNode, "SPORK_21_QUORUM_ALL_CONNECTED", 0)
	//Spork(targetSeedNode, "SPORK_23_QUORUM_POSE", 0)
	return nil
}

func run() {
	InitDockerClient()

	seedNode, err := CreateSeed()
	targetSeedNode := TargetFromHost(seedNode.Host)

	err = ActivateSporks(targetSeedNode)
	if err != nil {
		log.Fatal(err)
	}

	// Generate some initial blocks
	Generate(targetSeedNode, 200)

	numMasternodes := 10
	masternodes := <-CreateMasternodes(dockerClient, targetSeedNode, numMasternodes)

	for _, mnode := range masternodes {
		WaitForSync(mnode.Target)
	}

	initialChainlock := <-MineUntilChainlock(targetSeedNode)

	// Setup Mongo
	for _, mnode := range masternodes {
		mnode.MongoJob = SetupMongo(dockerClient, mnode)
	}
	for _, mnode := range masternodes {
		<-mnode.MongoJob
	}

	// Setup Drive
	for _, mnode := range masternodes {
		mnode.DriveJob = SetupDrive(dockerClient, mnode, initialChainlock)
	}
	for _, mnode := range masternodes {
		<-mnode.DriveJob
	}

	// Setup Tenderdash
	for _, mnode := range masternodes {
		mnode.TenderdashJob = SetupTenderdash(dockerClient, mnode, initialChainlock)
	}
	for _, mnode := range masternodes {
		<-mnode.TenderdashJob
	}

	// Wait for keypress
	log.Println("Press enter to shutdown...")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()

	// Cleanup
	log.Println("Stopping containers")

	for _, mnode := range masternodes {
		StopDrive(dockerClient, mnode)
		StopMongo(dockerClient, mnode)
	}

	dockerClient.ContainerStop(context.Background(), seedNode.Id, nil)
	for _, mnode := range masternodes {
		dockerClient.ContainerStop(context.Background(), mnode.ContainerId, nil)
	}
	log.Println("Removing containers")
	dockerClient.ContainersPrune(context.Background(), filters.Args{})
	log.Println("Removing volumes")
	dockerClient.VolumesPrune(context.Background(), filters.Args{})
	log.Println("Done.")
}
