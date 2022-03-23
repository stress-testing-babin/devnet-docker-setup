package main

import "log"

func WaitForBlockHeight(target *RpcTarget, height int) <-chan int {
	resChannel := make(chan int)

	go func() {
		res, err := target.NewRequest(
			"waitforblockheight",
			[]interface{}{
				height,
			},
		).Send()
		if err != nil {
			log.Fatal(err)
			resChannel <- -1
		} else {
			obj := res.(map[string]interface{})
			resChannel <- int(obj["height"].(float64))
		}
	}()

	return resChannel
}

func GetBlockHeight(target *RpcTarget) <-chan int {
	return WaitForBlockHeight(target, 0)
}
