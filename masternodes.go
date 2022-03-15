package main

import (
	"log"
	"time"
)

func ListMasternodes(target *RpcTarget) ([]string, error) {
	res, err := target.NewRequest(
		"protx",
		[]interface{}{
			"list",
			"valid",
		},
	).Send()
	if err != nil {
		return nil, err
	} else {
		masternodes := make([]string, 0)
		availableMasternodes := res.([]interface{})
		for _, mnode := range availableMasternodes {
			masternodes = append(masternodes, mnode.(string))
		}
		return masternodes, nil
	}
}

func MnIsSynced(target *RpcTarget) bool {
	res, err := target.NewRequest(
		"mnsync",
		[]interface{}{
			"status",
		},
	).Send()
	if err != nil {
		return false
	}
	status := res.(map[string]interface{})["IsSynced"].(bool)
	return status
}

func WaitForSync(target *RpcTarget) {
	printed := false
	for {
		status := MnIsSynced(target)
		if status {
			log.Println("Synced!")
			break
		} else {
			time.Sleep(10 * time.Second)
			if !printed {
				printed = true
				log.Println("Waiting for sync...")
			}
		}
	}
}
