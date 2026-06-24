package mcp

import (
	"kanbanai/internal/di"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewMCPHandler(container *di.Container) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "kanbanai-mcp",
		Version: "0.1.0",
	}, nil)

	RegisterTools(server, container)

	return mcp.NewSSEHandler(func(request *http.Request) *mcp.Server {
		return server
	}, nil)
}
