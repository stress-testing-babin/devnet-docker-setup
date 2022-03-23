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

func run() {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		fmt.Println("Unable to create docker client")
		log.Fatal(err)
	}

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
		log.Fatal(err)
	}
	targetSeedNode := TargetFromHost(seedNodeHost)

	// Generate some initial blocks
	Generate(targetSeedNode, 200)

	numMasternodes := 10
	masternodes := <-CreateMasternodes(dockerClient, targetSeedNode, numMasternodes)

	for _, mnode := range masternodes {
		WaitForSync(mnode.Target)
	}

	err = Spork(targetSeedNode, "SPORK_2_INSTANTSEND_ENABLED", 0)
	if err != nil {
		log.Fatal(err)
	}
	err = Spork(targetSeedNode, "SPORK_3_INSTANTSEND_BLOCK_FILTERING", 0)
	if err != nil {
		log.Fatal(err)
	}
	err = Spork(targetSeedNode, "SPORK_9_SUPERBLOCKS_ENABLED", 0)
	if err != nil {
		log.Fatal(err)
	}
	err = Spork(targetSeedNode, "SPORK_17_QUORUM_DKG_ENABLED", 0)
	if err != nil {
		log.Fatal(err)
	}
	err = Spork(targetSeedNode, "SPORK_19_CHAINLOCKS_ENABLED", 0)
	if err != nil {
		log.Fatal(err)
	}
	//Spork(targetSeedNode, "SPORK_21_QUORUM_ALL_CONNECTED", 0)
	//Spork(targetSeedNode, "SPORK_23_QUORUM_POSE", 0)

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

	dockerClient.ContainerStop(context.Background(), seedNodeId, nil)
	for _, mnode := range masternodes {
		dockerClient.ContainerStop(context.Background(), mnode.ContainerId, nil)
	}
	log.Println("Removing containers")
	dockerClient.ContainersPrune(context.Background(), filters.Args{})
	log.Println("Removing volumes")
	dockerClient.VolumesPrune(context.Background(), filters.Args{})
	log.Println("Done.")
}
