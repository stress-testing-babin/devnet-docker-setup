package orchestrator_shared

type CreateSeedResponse struct {
	SeedNodeHost string
}

type RpcProxyRequest struct {
	Host    string
	Command string
}

type RpcProxyResponse struct {
	Error   string
	Content string
}
