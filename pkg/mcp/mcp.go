package mcp

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/pkg/chat"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
)

const (
	anthropicAPIEndpoint = "https://api.anthropic.com/v1/messages"
	defaultModel         = "claude-3-sonnet-20240229"
)

// MCPClient represents the MCP client configuration
type MCPClient struct {
	apiKey     string
	client     *http.Client
	logger     logger.Logger
	repository chat.Repository
}

// NewMCPClient creates a new MCP client
func NewMCPClient(logger logger.Logger, repository chat.Repository) (*MCPClient, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY não encontrada nas variáveis de ambiente")
	}

	return &MCPClient{
		apiKey:     apiKey,
		client:     &http.Client{},
		logger:     logger,
		repository: repository,
	}, nil
}

// ContextData contains user context information
type ContextData struct {
	UserID   string
	TenantID string
	Role     string
}

// Message represents a chat message for the Anthropic API
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messageRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type messageResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// GetChatHistory retorna o histórico de chat para um usuário específico
func (m *MCPClient) GetChatHistory(ctx context.Context, userID string) ([]chat.Message, error) {
	// Buscar as últimas 50 mensagens
	messages, err := m.repository.GetUserHistory(ctx, userID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting chat history: %v", err)
	}
	return messages, nil
}

// ProcessWithContext processes a message with context and maintains chat history
func (m *MCPClient) ProcessWithContext(ctx context.Context, message string, contextData *ContextData) (string, error) {
	// Salvar mensagem do usuário
	userMessage := &chat.Message{
		UserID:  contextData.UserID,
		Role:    "user",
		Content: message,
	}
	if err := m.repository.SaveMessage(ctx, userMessage); err != nil {
		return "", fmt.Errorf("error saving user message: %v", err)
	}

	// Processar mensagem (aqui você pode adicionar a lógica de processamento real)
	response := fmt.Sprintf("Processando mensagem: %s", message)

	// Salvar resposta do assistente
	assistantMessage := &chat.Message{
		UserID:  contextData.UserID,
		Role:    "assistant",
		Content: response,
	}
	if err := m.repository.SaveMessage(ctx, assistantMessage); err != nil {
		return "", fmt.Errorf("error saving assistant message: %v", err)
	}

	return response, nil
}

// GetContextFromRequest creates context data from request information
func GetContextFromRequest(ctx *gin.Context) *ContextData {
	userID, _ := ctx.Get("user_id")
	tenantID, _ := ctx.Get("tenant_id")
	role, _ := ctx.Get("role")

	if userID == nil || tenantID == nil {
		return nil
	}

	return &ContextData{
		UserID:   userID.(string),
		TenantID: tenantID.(string),
		Role:     role.(string),
	}
}

func (c *MCPClient) DeleteChatHistory(ctx context.Context, userID string) error {
	if err := c.repository.DeleteUserHistory(ctx, userID); err != nil {
		return fmt.Errorf("error deleting chat history: %v", err)
	}
	return nil
}
