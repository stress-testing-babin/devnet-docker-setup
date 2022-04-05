package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/rpc"
	shared "orchestrator/orchestrator_shared"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
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

func (o *Orchestrator) GetCoreNodes(req *int, resp *[]string) error {
	containers, err := dockerClient.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(filters.KeyValuePair{Key: "name", Value: "_core$"}),
	})
	if err != nil {
		return err
	}
	for _, container := range containers {
		*resp = append(*resp, container.Names[0][1:])
	}
	return nil
}

// Proxy RPC to container
func (o *Orchestrator) RpcProxy(request *shared.RpcProxyRequest, resp *shared.RpcProxyResponse) error {
	target := TargetFromHost(request.Host)
	words := strings.Split(request.Command, " ")
	response, err := target.NewRequest(words[0], words[1:]).Send()
	buf, _ := json.MarshalIndent(response, "", "  ")
	resp.Content = string(buf)
	if err != nil {
		resp.Error = err.Error()
	} else {
		resp.Error = ""
	}
	return nil
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
