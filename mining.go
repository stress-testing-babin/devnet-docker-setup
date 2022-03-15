package main

import (
	"fmt"
	"log"
	"time"
)

func Generate(target *RpcTarget, blockCount int) error {
	res, err := target.NewRequest(
		"generate",
		map[string]interface{}{"nblocks": blockCount, "maxtries": 1000000000},
	).Send()
	if err != nil {
		return err
	} else {
		log.Println("Mined " + fmt.Sprint(len(res.([]interface{}))) + " blocks")
	}
	return nil
}

// Mine blocks until a given masternode appears as valid
func MineMasternodeConfirmation(target *RpcTarget, masternodeId string) error {
	for {
		masternodes, err := ListMasternodes(target)
		if err != nil {
			return err
		}

		for _, mnode := range masternodes {
			if mnode == masternodeId {
				log.Printf("Masternode %s available\n", target.Host)
				return nil
			}
		}
		Generate(target, 10)
	}
}

type Chainlock struct {
	Height int
	Hash   string
}

func MineUntilChainlock(target *RpcTarget) <-chan Chainlock {
	resChannel := make(chan Chainlock)

	go func() {
		printed := false
		for {
			res, err := target.NewRequest(
				"getbestchainlock",
				[]interface{}{},
			).Send()
			if err != nil {
				if !printed {
					printed = true
					log.Println(err)
					log.Println("Mining more blocks...")
				}
				Generate(target, 10)
				time.Sleep(5 * time.Second)
			} else {
				status := res.(map[string]interface{})
				height := status["height"].(float64)
				blockhash := status["blockhash"].(string)
				log.Printf("Chainlock created at height %f in block %s", height, blockhash)
				resChannel <- Chainlock{
					Height: int(height),
					Hash:   blockhash,
				}
				break
			}
		}
	}()
	return resChannel
}
