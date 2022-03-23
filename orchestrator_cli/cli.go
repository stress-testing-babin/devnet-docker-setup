package main

import (
	"fmt"
	"log"
	"net/rpc"
	shared "orchestrator/orchestrator_shared"
	"os"

	"github.com/c-bata/go-prompt"
)

func main() {
	p := prompt.New(executor, completer)
	p.Run()
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "createSeed", Description: "Create the seed node."},
		{Text: "quit", Description: "Exit prompt."},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}

func createSeed() {
	fmt.Println("Creating seed node")
	serverAddress := "localhost"
	client, err := rpc.DialHTTP("tcp", serverAddress+":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	var resp shared.CreateSeedResponse
	err = client.Call("Orchestrator.CreateSeed", 0, &resp)
	if err != nil {
		log.Fatal("call error:", err)
	} else {
		fmt.Println("Host is :" + resp.SeedNodeHost)
	}

}

func executor(cmd string) {
	if cmd == "quit" {
		fmt.Println("Exiting...")
		os.Exit(0)
	} else if cmd == "createSeed" {
		createSeed()
	} else {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}
}
