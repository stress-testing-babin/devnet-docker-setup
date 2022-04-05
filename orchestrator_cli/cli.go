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

var coreNodes []string

func completer(d prompt.Document) []prompt.Suggest {
	options := []prompt.Suggest{
		{Text: "createSeed", Description: "Create the seed node."},
		{Text: "activateSporks", Description: "Activate the necessary sporks."},
		{Text: "connect", Description: "Connect to orchestrator."},
		{Text: "getCoreNodes", Description: "Get all core nodes."},
		{Text: "rpc", Description: "Proxy an RPC command to a node: rpc <host> <command>"},
		{Text: "quit", Description: "Exit prompt."},
	}

	if strings.HasPrefix(d.Text, "rpc ") {
		host := strings.TrimPrefix(d.Text, "rpc ")

		if len(strings.TrimSpace(host)) > 0 && strings.HasSuffix(host, " ") {
			options = []prompt.Suggest{
				{Text: "getmininginfo", Description: "Get mining information."},
			}
		} else {
			options = []prompt.Suggest{}
			for _, coreNode := range coreNodes {
				options = append(options, prompt.Suggest{Text: coreNode})
			}
		}
	} else {
		options = prompt.FilterHasPrefix(options, d.GetWordBeforeCursor(), true)
	}

	return options
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

func GetCoreNodes() {
	fmt.Println("Retrieving core nodes...")
	var resp []string
	var args int = 0
	err := client.Call("Orchestrator.GetCoreNodes", &args, &resp)
	if err != nil {
		log.Print("GetCoreNodes error: ", err)
	} else {
		coreNodes = resp
		for _, node := range resp {
			fmt.Println(node)
		}
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

func RpcProxy(req shared.RpcProxyRequest) {
	var resp shared.RpcProxyResponse
	err := client.Call("Orchestrator.RpcProxy", &req, &resp)
	if err != nil {
		fmt.Println("Error: ", err)
	} else {
		fmt.Println(resp.Content)
	}
}

func executor(cmd string) {
	if cmd == "quit" {
		fmt.Println("Exiting...")
		os.Exit(0)
	} else if cmd == "connect" {
		Connect()
	} else if strings.HasPrefix(cmd, "rpc ") {
		params := strings.TrimPrefix(cmd, "rpc ")
		parts := strings.SplitN(params, " ", 2)
		host := parts[0]
		rpcCommand := parts[1]
		req := shared.RpcProxyRequest{
			Host:    host,
			Command: rpcCommand,
		}
		RpcProxy(req)
	} else if cmd == "createSeed" {
		CreateSeed()
	} else if cmd == "getCoreNodes" {
		GetCoreNodes()
	} else if strings.HasPrefix(cmd, "activateSporks") {
		ActivateSporks(strings.TrimSpace(strings.TrimPrefix(cmd, "activateSporks")))
	} else {
		fmt.Printf("Unknown command '%s'\n", cmd)
	}
}
