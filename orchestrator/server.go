package main

import (
	"fmt"
	"net/http"
	"net/rpc"
	shared "orchestrator/orchestrator_shared"
)

type Orchestrator int

func (o *Orchestrator) CreateSeed(args *int, resp *shared.CreateSeedResponse) error {
	fmt.Println("CreateSeed")
	resp.SeedNodeHost = "MeinTollerHost"
	return nil
}

func main() {
	orchestrator := new(Orchestrator)
	rpc.Register(orchestrator)
	rpc.HandleHTTP()

	err := http.ListenAndServe(":1234", nil)
	if err != nil {
		fmt.Println(err.Error())
	}
}
