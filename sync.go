package main

import "log"

func WaitForNodeSync(target *RpcTarget, others []*RpcTarget) {
	targetHeight := <-GetBlockHeight(target)
	for _, other := range others {
		height := <-WaitForBlockHeight(other, targetHeight)
		log.Printf("Node %s has height %d\n", other.Host, height)
	}
}
