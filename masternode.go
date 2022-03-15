package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
)

const P2P_PORT = "20001"
const RPC_PORT = "20002"

type ContainerConfig struct {
	Target           *RpcTarget
	Volume           *types.Volume
	EndpointSettings *network.EndpointSettings
	ContainerId      string
}

func createDashd(cli *client.Client, seedTarget *RpcTarget, name string) <-chan ContainerConfig {
	resChannel := make(chan ContainerConfig)

	go func() {
		// Create dashd container
		masternodeHost, masternodeContainerId, nodeVolume, endpointSettings, err := CreateFullnode(
			cli,
			name,
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

		masternodeTarget := NewRpcTarget(masternodeHost, RPC_PORT, "dashrpc", "rpcpassword")
		masternodeTarget.WaitUntilAvailable() // Wait until dashd is available over RPC

		// Connect new node to seed node
		connectResChan := ConnectNode(masternodeTarget, seedTarget.Host, P2P_PORT)
		err = <-connectResChan
		if err != nil {
			log.Fatal(err)
		}

		resChannel <- ContainerConfig{
			Target:           masternodeTarget,
			Volume:           nodeVolume,
			EndpointSettings: endpointSettings,
			ContainerId:      masternodeContainerId,
		}
	}()

	return resChannel
}

type CollateralInfo struct {
	Address string
	TxId    string
}

func createCollateral(masternodeTarget *RpcTarget, seedTarget *RpcTarget) <-chan CollateralInfo {
	resChannel := make(chan CollateralInfo)

	go func() {
		collateralAddress, err := GenerateNewAddress(masternodeTarget)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Generated collateral address: %s\n", collateralAddress)

		collateralTxId, err := SeedSendFunds(seedTarget, collateralAddress, 1000.0)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Sent collateral in transaction: " + collateralTxId)
		resChannel <- CollateralInfo{
			Address: collateralAddress,
			TxId:    collateralTxId,
		}
	}()

	return resChannel
}

type MasternodeConfig struct {
	Name                  string
	Host                  string
	Volume                *types.Volume
	Target                *RpcTarget
	EndpointSettings      *network.EndpointSettings
	ContainerId           string
	Collateral            CollateralInfo
	PublicKey             string
	PrivateKey            string
	OwnerKeyAddr          string
	VotingKeyAddr         string
	PayoutAddr            string
	FeeSourceAddr         string
	MasternodeId          string
	MongoContainerId      string
	MongoVolume           *types.Volume
	DriveContainerId      string
	DriveVolume           *types.Volume
	TenderdashContainerId string
	TenderdashVolume      *types.Volume
	// Jobs
	ContainerJob  <-chan ContainerConfig
	CollateralJob <-chan CollateralInfo
	ProRegJob     <-chan error
	ConnectionJob <-chan error
	MongoJob      <-chan error
	DriveJob      <-chan error
	TenderdashJob <-chan error
}

func ProRegTx(cli *client.Client, seedTarget *RpcTarget, mnode *MasternodeConfig) <-chan error {
	resChannel := make(chan error)

	go func() {
		log.Println("Requesting container IP")
		container, err := cli.ContainerInspect(context.Background(), mnode.ContainerId)
		if err != nil {
			log.Fatal(err)
		}
		containerIp := container.NetworkSettings.Networks["devnet"].IPAddress
		log.Printf("Container %s IP is %s\n", mnode.Target.Host, containerIp)

		log.Println("Prepare ProRegTx")
		var signMessage string
		var serializedTx string
		for {
			res, err := mnode.Target.NewRequest(
				"protx",
				[]interface{}{
					"register_prepare",
					mnode.Collateral.TxId,
					1, // Collateral Index
					fmt.Sprintf(containerIp) + ":" + P2P_PORT, // TODO: Static IPs?
					mnode.OwnerKeyAddr,
					mnode.PublicKey,
					mnode.VotingKeyAddr,
					0,
					mnode.PayoutAddr,
					mnode.FeeSourceAddr,
				},
			).Send()
			if err != nil {
				log.Println(err)
				Generate(seedTarget, 1)
				time.Sleep(5 * time.Second)
				continue
			} else {
				signMessage = res.(map[string]interface{})["signMessage"].(string)
				serializedTx = res.(map[string]interface{})["tx"].(string)
				break
			}
		}

		log.Println("Sign ProRegTx transaction")
		res, err := mnode.Target.NewRequest(
			"signmessage",
			[]interface{}{
				mnode.Collateral.Address,
				signMessage,
			},
		).Send()
		var messageSignature string
		if err != nil {
			log.Fatal(err)
		} else {
			messageSignature = res.(string)
		}

		log.Println("Submit signed ProRegTx transaction")
		res, err = mnode.Target.NewRequest(
			"protx",
			[]interface{}{
				"register_submit",
				serializedTx,
				messageSignature,
			},
		).Send()
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res)
			mnode.MasternodeId = res.(string)
		}
		resChannel <- nil
	}()
	return resChannel
}

func CreateMasternodes(cli *client.Client, seedTarget *RpcTarget, nodeCount int) <-chan []*MasternodeConfig {
	resChannel := make(chan []*MasternodeConfig)

	go func() {
		masternodes := make([]*MasternodeConfig, nodeCount)

		// Generate Names
		for i := 0; i < nodeCount; i++ {
			masternodes[i] = &MasternodeConfig{}
			masternodes[i].Name = fmt.Sprintf("go_m%d", i)
			masternodes[i].ContainerJob = createDashd(cli, seedTarget, masternodes[i].Name)
		}

		for i := 0; i < nodeCount; i++ {
			conf := <-masternodes[i].ContainerJob
			masternodes[i].Target = conf.Target
			masternodes[i].ContainerId = conf.ContainerId
			masternodes[i].Volume = conf.Volume
			masternodes[i].EndpointSettings = conf.EndpointSettings
		}

		for i := 0; i < nodeCount; i++ {
			masternodes[i].CollateralJob = createCollateral(masternodes[i].Target, seedTarget)
		}
		for i := 0; i < nodeCount; i++ {
			masternodes[i].Collateral = <-masternodes[i].CollateralJob
		}

		for i := 0; i < nodeCount; i++ {
			blsPublicKey, blsPrivateKey := GenerateBlsKeypair(masternodes[i].Target)
			masternodes[i].PublicKey = blsPublicKey
			masternodes[i].PrivateKey = blsPrivateKey

			ownerKeyAddr, err := GenerateNewAddress(masternodes[i].Target)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Generated Owner Address %s\n", ownerKeyAddr)
			masternodes[i].OwnerKeyAddr = ownerKeyAddr

			votingKeyAddr, err := GenerateNewAddress(masternodes[i].Target)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Generated Voting Address %s\n", votingKeyAddr)
			masternodes[i].VotingKeyAddr = votingKeyAddr

			payoutAddress, err := GenerateNewAddress(masternodes[i].Target)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Generated Payout Address %s\n", payoutAddress)
			masternodes[i].PayoutAddr = payoutAddress

			feeSourceAddress, err := GenerateNewAddress(masternodes[i].Target)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Generated Fee Source Address %s\n", feeSourceAddress)
			masternodes[i].FeeSourceAddr = feeSourceAddress

			txid, err := SeedSendFunds(seedTarget, payoutAddress, 10.0)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Sent funds for transaction fees to payout address in %s\n", txid)

			txid, err = SeedSendFunds(seedTarget, feeSourceAddress, 10.0)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Sent funds for transaction fees to payout address in %s\n", txid)
		}

		log.Println("Mining funds for confirmation")
		/*err = MineUntilBalanceReached(seedTarget, payoutAddress, 100.0)
		if err != nil {
			log.Fatal(err)
		}*/
		err := Generate(seedTarget, 110)
		if err != nil {
			log.Fatal(err)
		}

		// Make sure all the nodes have the same chain height
		masternodeTargets := make([]*RpcTarget, len(masternodes))
		for i := 0; i < len(masternodeTargets); i++ {
			masternodeTargets[i] = masternodes[i].Target
		}
		WaitForNodeSync(seedTarget, masternodeTargets)

		for i := 0; i < nodeCount; i++ {
			masternodes[i].ProRegJob = ProRegTx(cli, seedTarget, masternodes[i])
		}
		for i := 0; i < nodeCount; i++ {
			err = <-masternodes[i].ProRegJob
			if err != nil {
				log.Fatal(err)
			}
		}

		log.Println("Mining funds for confirmation")
		err = Generate(seedTarget, 100)
		if err != nil {
			log.Fatal(err)
		}

		// Set masternodeblsprivkey
		// Do this sequentially so there are no address swaps
		for i := 0; i < nodeCount; i++ {
			// Stop container
			err = cli.ContainerStop(context.Background(), masternodes[i].ContainerId, nil)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Stopped container %s\n", masternodes[i].Name)

			// Remove container
			err = cli.ContainerRemove(context.Background(), masternodes[i].ContainerId, types.ContainerRemoveOptions{})
			if err != nil {
				log.Fatal(err)
			}
			masternodes[i].ContainerId = ""
			log.Printf("Removed container %s\n", masternodes[i].Name)

			// Recreate fullnode with masternodeBlsKey set
			masternodeHost, masternodeContainerId, nodeVolume, endpointSettings, err := CreateFullnode(
				cli,
				masternodes[i].Name,
				strslice.StrSlice{
					"dashd",
					"-masternodeblsprivkey=" + masternodes[i].PrivateKey,
					"-seednode=" + seedTarget.Host + ":" + P2P_PORT,
					"-llmqdevnetparams=" + DEVNET_PARAMS,
					"-llmqchainlocks=llmq_devnet",
					"-llmqinstantsend=llmq_devnet",
				},
				masternodes[i].Volume,
				masternodes[i].EndpointSettings,
			)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Recreated container %s\n", masternodes[i].Name)

			masternodes[i].Target.Host = masternodeHost
			masternodes[i].ContainerId = masternodeContainerId
			masternodes[i].Volume = nodeVolume
			masternodes[i].EndpointSettings = endpointSettings
		}

		for i := 0; i < nodeCount; i++ {
			masternodes[i].Target.WaitUntilAvailable() // Wait until dashd is available again
		}

		// Reconnect new masternode to seed node
		/*for i := 0; i < nodeCount; i++ {
			masternodes[i].ConnectionJob = ConnectNode(masternodes[i].Target, seedTarget.Host, P2P_PORT)
		}
		for i := 0; i < nodeCount; i++ {
			err = <-masternodes[i].ConnectionJob
			if err != nil {
				log.Fatal(err)
			}
		}*/

		/*log.Println("Requesting container IP")
		container, err = cli.ContainerInspect(context.Background(), masternodeContainerId)
		if err != nil {
			log.Fatal(err)
		}
		newContainerIP := container.NetworkSettings.Networks["devnet"].IPAddress
		if newContainerIP != containerIp {
			log.Fatalf("Container %s has switched its IP from %s to %s!\n", masternodeTarget.Host, containerIp, newContainerIP)
		}

		log.Printf("Masternode %s has IP %s\n", masternodeHost, newContainerIP)*/

		/*log.Println("List available masternodes")
		masternodes, err := ListMasternodes(seedTarget)
		success := false
		for _, mnode := range masternodes {
			if mnode == masternodeId {
				log.Println("SUCCESS!")
				log.Println("Masternode was successfully registered!")
				success = true
			}
		}
		if !success {
			log.Println("ERROR")
			log.Println("New masternode does not appear in available node list!")
		}*/
		//MineMasternodeConfirmation(seedTarget, masternodeId)
		err = Generate(seedTarget, 110)
		if err != nil {
			log.Fatal(err)
		}

		resChannel <- masternodes
	}()

	return resChannel
}
