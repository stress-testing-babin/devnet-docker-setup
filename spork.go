package main

import "log"

func Spork(target *RpcTarget, sporkName string, value int) error {
	_, err := target.NewRequest(
		"spork",
		[]interface{}{
			sporkName,
			value,
		},
	).Send()
	if err != nil {
		return err
	}
	log.Printf("%s set to: %d\n", sporkName, value)
	return nil
}
