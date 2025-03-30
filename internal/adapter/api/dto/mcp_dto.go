package dto

import (
	"github.com/hugohenrick/erp-supermercado/pkg/chat"
)

// MCPMessageRequest representa uma requisição de mensagem para o MCP
type MCPMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

// MCPMessageResponse representa a resposta do MCP
type MCPMessageResponse struct {
	Response string         `json:"response"`
	History  []chat.Message `json:"history"`
}

// NewMCPMessageResponse cria uma nova resposta MCP com histórico
func NewMCPMessageResponse(response string, history []chat.Message) MCPMessageResponse {
	return MCPMessageResponse{
		Response: response,
		History:  history,
	}
}
