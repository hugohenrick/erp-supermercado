package models

import (
	"github.com/hugohenrick/erp-supermercado/pkg/mcp"
)

// MCPRequest representa uma solicitação de processamento de mensagem
type MCPRequest struct {
	Message string `json:"message" binding:"required"`
}

// MCPResponse representa a resposta a uma solicitação de processamento de mensagem
type MCPResponse struct {
	Message string `json:"message"`
}

// MCPHistoryResponse representa a resposta para a solicitação de histórico de mensagens
type MCPHistoryResponse struct {
	Messages []mcp.Message `json:"messages"`
}
