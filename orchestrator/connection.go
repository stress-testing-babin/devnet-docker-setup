package main

import (
	"log"
	"time"
)

// Connects node and waits until it is added to the peer list
func ConnectNode(target *RpcTarget, nodeHost string, nodePort string) <-chan error {
	resChannel := make(chan error)

	go func() {
		AddNode(target, nodeHost, nodePort) // Error ignored, is usually "Node already added"

		printed := false
		for {
			nodes, err := GetPeerInfo(target)
			if err != nil {
				resChannel <- err
				return
			}
			success := false
			for _, node := range nodes {
				addr := node.(map[string]interface{})["addr"].(string)
				addrlocal := node.(map[string]interface{})["addrlocal"].(string)
				if addr == nodeHost+":"+nodePort || addrlocal == nodeHost+":"+nodePort {
					success = true
				}
			}
			if success {
				log.Println("Connected!")
				break
			} else {
				time.Sleep(time.Second)
				if !printed {
					printed = true
					log.Println("Waiting for node connection...")
				}
			}
		}
		resChannel <- nil
	}()

	return resChannel
}

func AddNode(target *RpcTarget, nodeHost string, nodePort string) error {
	_, err := target.NewRequest(
		"addnode",
		[]interface{}{
			nodeHost + ":" + nodePort,
			"add",
		},
	).Send()
	return err
}

func GetPeerInfo(target *RpcTarget) ([]interface{}, error) {
	res, err := target.NewRequest(
		"getpeerinfo",
		[]interface{}{},
	).Send()
	if err != nil {
		return nil, err
	} else {
		nodes := res.([]interface{})
		return nodes, nil
	}
}
