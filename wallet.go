package main

import (
	"fmt"
	"log"
	"math"
)

const BLOCK_REWARD = 500.0

func GetBalance(target *RpcTarget) (float64, error) {
	balance, err := target.NewRequest(
		"getbalance",
		map[string]interface{}{},
	).Send()
	if err != nil {
		return -1, err
	}
	return balance.(float64), nil
}

func MineUntilBalanceReached(target *RpcTarget, addr string, targetBalance float64) error {
	for {
		balance, err := GetAddressBalance(target, addr)
		if err != nil {
			return err
		}
		if balance < targetBalance {
			err = Generate(target, 10)
			if err != nil {
				return err
			}
		} else {
			return nil
		}
	}
}

func GetAddressBalance(target *RpcTarget, addr string) (float64, error) {
	res, err := target.NewRequest(
		"getaddressbalance",
		[]interface{}{
			addr,
		},
	).Send()
	if err != nil {
		return -1, err
	} else {
		duffBalance := res.(map[string]interface{})["balance"].(float64)
		dashBalance := duffBalance / 100000000
		return dashBalance, nil
	}
}

func GetTransaction(target *RpcTarget, txid string) error {
	res, err := target.NewRequest(
		"getrawtransaction",
		[]interface{}{txid, 1},
	).Send()
	if err != nil {
		return err
	} else {
		fmt.Println(res)
	}
	return nil
}

func SeedSendFunds(seedTarget *RpcTarget, destAddr string, amount float64) (string, error) {
	balance, err := GetBalance(seedTarget)
	if err != nil {
		return "Unknown", err
	}
	log.Printf("Balance: %f, required is %f\n", balance, amount)

	if balance <= amount {
		missingBalance := amount - balance
		blockCount := int(math.Ceil(missingBalance / BLOCK_REWARD))
		log.Printf("Missing %f dash, mining %d blocks\n", missingBalance, blockCount)

		err = Generate(seedTarget, blockCount)
		if err != nil {
			return "Unknown", err
		} else {
			log.Println("Mined " + fmt.Sprint(blockCount) + " blocks")
		}
	}

	res, err := seedTarget.NewRequest(
		"sendtoaddress",
		map[string]interface{}{
			"address": destAddr,
			"amount":  amount,
		},
	).Send()
	if err != nil {
		return "Unknown", err
	} else {
		txid := res.(string)
		return txid, nil
	}
}

func GenerateNewAddress(target *RpcTarget) (string, error) {
	res, err := target.NewRequest(
		"getnewaddress",
		map[string]interface{}{},
	).Send()
	if err == nil {
		address := res.(string)
		return address, nil
	} else {
		return "Unknown", err
	}
}

func GenerateBlsKeypair(target *RpcTarget) (string, string) {
	log.Println("Generating BLS key pair")
	var blsPublicKey string
	var blsPrivateKey string
	res, err := target.NewRequest(
		"bls",
		[]interface{}{
			"generate",
		},
	).Send()
	if err == nil {
		blsPrivateKey = res.(map[string]interface{})["secret"].(string)
		blsPublicKey = res.(map[string]interface{})["public"].(string)
	} else {
		log.Fatal(err)
	}
	log.Println("Public: " + blsPublicKey)
	log.Println("Private: " + blsPrivateKey)
	return blsPublicKey, blsPrivateKey
}
