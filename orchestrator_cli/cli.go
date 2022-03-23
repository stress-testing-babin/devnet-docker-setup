package main

import (
	"fmt"
	"log"
	"net/rpc"
	shared "orchestrator/orchestrator_shared"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
)

var client *rpc.Client

func main() {
	p := prompt.New(executor, completer)
	p.Run()
}

func completer(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "createSeed", Description: "Create the seed node."},
		{Text: "activateSporks", Description: "Activate the necessary sporks."},
		{Text: "connect", Description: "Connect to orchestrator."},
		{Text: "quit", Description: "Exit prompt."},
	}
	return prompt.FilterFuzzy(s, d.GetWordBeforeCursor(), true)
}

func Connect() {
	serverAddress := "localhost"
	port := "1234"
	fmt.Printf("Connecting to orchestrator at %s:%s...\n", serverAddress, port)
	var err error
	client, err = rpc.DialHTTP("tcp", serverAddress+":"+port)
	if err != nil {
		fmt.Println("Connection error:", err)
	} else {
		fmt.Println("Connection established.")
	}
}

func CreateSeed() {
	fmt.Println("Creating seed node...")
	var resp shared.CreateSeedResponse
	var args int = 0
	err := client.Call("Orchestrator.CreateSeed", &args, &resp)
	if err != nil {
		log.Print("CreateSeed error: ", err)
	} else {
		fmt.Println("Host is:" + resp.SeedNodeHost)
	}
}

func ActivateSporks(host string) {
	fmt.Println("Activating sporks...")
	var resp bool
	err := client.Call("Orchestrator.ActivateSporks", &host, &resp)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	success := "successful"
	if !resp {
		success = "unsuccessful"
	}
	fmt.Println("Spork activation " + success)
}

func executor(cmd string) {
	if cmd == "quit" {
		fmt.Println("Exiting...")
		os.Exit(0)
	} else if cmd == "connect" {
		Connect()
	} else if cmd == "createSeed" {
		CreateSeed()
	} else if strings.HasPrefix(cmd, "activateSporks") {
		ActivateSporks(strings.TrimSpace(strings.TrimPrefix(cmd, "activateSporks")))
	} else {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}
}
