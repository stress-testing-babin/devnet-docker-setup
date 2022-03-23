package main

import (
	"fmt"
	"net/http"
	"net/rpc"
	shared "orchestrator/orchestrator_shared"
)

type Orchestrator int

func (o *Orchestrator) CreateSeed(args *int, resp *shared.CreateSeedResponse) error {
	seedNode, err := CreateSeed()
	resp.SeedNodeHost = seedNode.Host
	return err
}

func (o *Orchestrator) ActivateSporks(host *string, resp *bool) error {
	target := TargetFromHost(*host)
	err := ActivateSporks(target)
	*resp = err == nil
	return err
}

func main() {
	InitDockerClient()

	orchestrator := new(Orchestrator)
	rpc.Register(orchestrator)
	rpc.HandleHTTP()

	fmt.Println("Listening on 1234.")
	err := http.ListenAndServe(":1234", nil)
	if err != nil {
		fmt.Println(err.Error())
	}
}
