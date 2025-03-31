package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/pkg/chat"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/mcp/intent"
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
	System    string    `json:"system,omitempty"`
}

type messageResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GetChatHistory retorna o histórico de chat para um usuário específico
func (m *MCPClient) GetChatHistory(ctx context.Context, userID string) ([]chat.Message, error) {
	// Buscar as últimas 50 mensagens
	messages, err := m.repository.GetUserHistory(ctx, userID, 50, 0)
	if err != nil {
		return nil, fmt.Errorf("error getting chat history: %v", err)
	}

	// We return messages as-is, they're already in reverse chronological order (newest first)
	// The front-end is responsible for rendering them in the correct order
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

	// Recuperar histórico de mensagens para fornecer contexto à API
	history, err := m.repository.GetUserHistory(ctx, contextData.UserID, 10, 0)
	if err != nil {
		m.logger.Error("Erro ao recuperar histórico de mensagens", "error", err)
		// Continue mesmo com erro no histórico
	}

	// Preparar as mensagens para a API da Anthropic
	messages := []Message{}

	// Criar mensagem de sistema para contexto
	systemPrompt := fmt.Sprintf("Você é um assistente para o ERP de gestão de lojas e supermercados. Seu nome é Angie(feminino)! "+
		"Você está conversando com um usuário do sistema com ID %s, "+
		"que pertence ao tenant %s com a função %s. "+
		"Forneça assistência focada em supermercados, inventário, vendas e gestão.",
		contextData.UserID, contextData.TenantID, contextData.Role)

	// Adicionar histórico recente (inverter a ordem para ter as mensagens mais antigas primeiro)
	// The database returns newest first, so we reverse it for chronological order for the API
	for i := len(history) - 1; i >= 0; i-- {
		msg := history[i]
		// Pular mensagens do sistema para evitar confusão
		if msg.Role == "system" {
			continue
		}
		messages = append(messages, Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Adicionar a mensagem atual se não estiver no histórico
	if message != "" && (len(history) == 0 || history[0].Content != message) {
		messages = append(messages, Message{
			Role:    "user",
			Content: message,
		})
	}

	// Criar a requisição para a API da Anthropic
	reqBody := messageRequest{
		Model:     defaultModel,
		MaxTokens: 1000,
		Messages:  messages,
		System:    systemPrompt,
	}

	m.logger.Info("Enviando requisição para API Anthropic",
		"model", reqBody.Model,
		"numMessages", len(reqBody.Messages))

	// Serializar a requisição
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		m.logger.Error("Erro ao serializar requisição", "error", err)
		return "Erro ao processar sua mensagem", err
	}

	// Log da requisição para debug
	m.logger.Debug("Payload da requisição", "json", string(reqJSON))

	// Criar o request HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIEndpoint, bytes.NewBuffer(reqJSON))
	if err != nil {
		m.logger.Error("Erro ao criar requisição HTTP", "error", err)
		return "Erro ao se comunicar com o assistente", err
	}

	// Configurar headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", m.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "messages-2023-12-15")
	req.Header.Set("Accept", "application/json")

	// Enviar a requisição
	resp, err := m.client.Do(req)
	if err != nil {
		m.logger.Error("Erro na chamada da API", "error", err)
		return "Erro na comunicação com o serviço de IA", err
	}
	defer resp.Body.Close()

	// Ler o body da resposta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.logger.Error("Erro ao ler resposta", "error", err)
		return "Erro ao processar resposta do assistente", err
	}

	// Log da resposta para debug
	m.logger.Debug("Resposta da API", "statusCode", resp.StatusCode, "body", string(respBody))

	// Verificar código de status
	if resp.StatusCode != http.StatusOK {
		m.logger.Error("API retornou erro",
			"status", resp.Status,
			"body", string(respBody))
		return fmt.Sprintf("Erro no serviço de IA (código %d)", resp.StatusCode),
			fmt.Errorf("API error: %s", resp.Status)
	}

	// Deserializar a resposta
	var apiResp struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		StopSequence string `json:"stop_sequence"`
		Usage        struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		m.logger.Error("Erro ao deserializar resposta", "error", err, "body", string(respBody))
		return "Erro ao interpretar resposta do assistente", err
	}

	// Extrair o texto da resposta
	response := ""
	for _, content := range apiResp.Content {
		if content.Type == "text" {
			response += content.Text
		}
	}

	if response == "" {
		m.logger.Error("Resposta vazia da API", "body", string(respBody))
		response = "Não foi possível gerar uma resposta. Tente novamente."
	}

	// Log de informações úteis sobre a resposta
	m.logger.Info("Resposta gerada com sucesso",
		"model", apiResp.Model,
		"input_tokens", apiResp.Usage.InputTokens,
		"output_tokens", apiResp.Usage.OutputTokens,
		"stop_reason", apiResp.StopReason)

	// Salvar resposta do assistente
	assistantMessage := &chat.Message{
		UserID:  contextData.UserID,
		Role:    "assistant",
		Content: response,
	}
	if err := m.repository.SaveMessage(ctx, assistantMessage); err != nil {
		m.logger.Error("Erro ao salvar mensagem do assistente", "error", err)
		// Continue mesmo com erro ao salvar
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

// MCP representa o processador de linguagem natural
type MCP struct {
	Logger         logger.Logger
	APIKey         string
	Model          string
	MessageHistory map[string][]Message
	IntentManager  *intent.IntentManager
	// Sessões de intenção - mapeia sessões para dados de estado
	IntentSessions map[string]*intent.FlowState
}

// NewMCP cria uma nova instância do MCP
func NewMCP(log logger.Logger, apiKey string, model string) *MCP {
	return &MCP{
		Logger:         log,
		APIKey:         apiKey,
		Model:          model,
		MessageHistory: make(map[string][]Message),
		IntentManager:  intent.NewIntentManager(log),
		IntentSessions: make(map[string]*intent.FlowState),
	}
}

// getSessionKey gera uma chave de sessão para um usuário e tenant
func (m *MCP) getSessionKey(tenantID, userID string) string {
	return fmt.Sprintf("%s:%s", tenantID, userID)
}

// RegisterHandler registra um handler de intenção no MCP
func (m *MCP) RegisterHandler(handler intent.IntentHandler) {
	m.IntentManager.RegisterHandler(handler)
}

// ProcessWithContext processa uma mensagem com contexto do usuário
func (m *MCP) ProcessWithContext(ctx context.Context, userMessage string, userID string, tenantID string, userRole string, locale string) (*Message, error) {
	// Gerar ID da sessão
	sessionID := m.getSessionKey(tenantID, userID)

	m.Logger.Info("Processing message with context",
		"user_message", userMessage,
		"user_id", userID,
		"tenant_id", tenantID,
		"has_intent_session", m.IntentSessions[sessionID] != nil)

	// DIRECT INTERCEPT - Bypass Claude completely for customer creation
	// Check if this is specifically a customer creation message
	lowerMsg := strings.ToLower(userMessage)
	// Immediate bypass for customer creation messages
	if strings.Contains(lowerMsg, "cadastre um novo cliente") &&
		(strings.Contains(userMessage, "Nome:") || strings.Contains(userMessage, "nome:")) &&
		(strings.Contains(userMessage, "CPF:") || strings.Contains(userMessage, "cpf:")) {

		m.Logger.Warn("BYPASSING CLAUDE API for customer creation message",
			"message", userMessage)

		// Return a dummy message saying we detected a customer creation attempt
		return &Message{
			Role:    "assistant",
			Content: "⚠️ Detectei uma solicitação para cadastro de cliente, mas não consegui processá-la adequadamente. Por favor, verifique o formato dos dados e tente novamente.",
		}, nil
	}

	isCustomerCreation := strings.Contains(lowerMsg, "cadastr") && strings.Contains(lowerMsg, "cliente") ||
		strings.Contains(lowerMsg, "cri") && strings.Contains(lowerMsg, "cliente") ||
		(strings.Contains(lowerMsg, "nome:") &&
			(strings.Contains(lowerMsg, "cpf:") || strings.Contains(lowerMsg, "cnpj:") ||
				strings.Contains(lowerMsg, "documento:")))

	// Check for confirmation message in an existing session
	isConfirmation := len(userMessage) < 20 && (strings.Contains(lowerMsg, "sim") ||
		strings.Contains(lowerMsg, "confirme") ||
		strings.Contains(lowerMsg, "confirmar") ||
		strings.Contains(lowerMsg, "confirmo") ||
		strings.Contains(lowerMsg, "ok") ||
		strings.Contains(lowerMsg, "pode cadastrar") ||
		strings.Contains(lowerMsg, "correto"))

	// FORCE intent processing for customer creation or confirmation messages
	if isCustomerCreation || isConfirmation || m.IntentSessions[sessionID] != nil {
		m.Logger.Info("INTERCEPTING message for direct intent processing",
			"is_customer_creation", isCustomerCreation,
			"is_confirmation", isConfirmation,
			"has_active_session", m.IntentSessions[sessionID] != nil)

		// Build context data
		intentCtx := intent.ContextData{
			UserID:   userID,
			TenantID: tenantID,
			Role:     userRole,
			Locale:   locale,
		}

		// If session exists, inject it
		if state, exists := m.IntentSessions[sessionID]; exists {
			m.Logger.Info("Injecting existing session state",
				"session_id", sessionID,
				"state", state.State,
				"intent", state.PendingIntent.Name)
			m.IntentManager.InjectSession(sessionID, state)
		}

		// Process with intent manager
		result, err := m.IntentManager.ProcessMessage(ctx, userMessage, intentCtx)

		// Save updated session state
		newState := m.IntentManager.ExtractSession(sessionID)
		if newState != nil {
			m.IntentSessions[sessionID] = newState
			m.Logger.Info("Updated session state",
				"session_id", sessionID,
				"state", newState.State)
		} else if _, exists := m.IntentSessions[sessionID]; exists {
			delete(m.IntentSessions, sessionID)
			m.Logger.Info("Cleared completed session", "session_id", sessionID)
		}

		if err == nil && result != nil {
			// Successful processing
			if result.Success {
				m.Logger.Info("Intent processing successful",
					"message", result.Message)

				// Save to message history
				userMsg := Message{
					Role:    "user",
					Content: userMessage,
				}
				systemMsg := Message{
					Role:    "assistant",
					Content: result.Message,
				}

				// Update history
				history := m.MessageHistory[sessionID]
				if history == nil {
					history = make([]Message, 0)
				}
				history = append(history, userMsg, systemMsg)
				m.MessageHistory[sessionID] = history

				return &systemMsg, nil
			} else if result.Message != "" {
				// Intent identified but action failed or needs more info
				m.Logger.Info("Intent processing returned a message",
					"message", result.Message,
					"success", result.Success)

				// Save to message history
				userMsg := Message{
					Role:    "user",
					Content: userMessage,
				}
				systemMsg := Message{
					Role:    "assistant",
					Content: result.Message,
				}

				// Update history
				history := m.MessageHistory[sessionID]
				if history == nil {
					history = make([]Message, 0)
				}
				history = append(history, userMsg, systemMsg)
				m.MessageHistory[sessionID] = history

				return &systemMsg, nil
			}
		} else if err != nil {
			// Log error for debugging
			m.Logger.Error("Error in intent processing", "error", err)
		} else {
			m.Logger.Warn("Intent processing returned nil result")
		}
	}

	// Verificar se é uma mensagem que pode ser processada como uma intenção
	intentCtx := intent.ContextData{
		UserID:   userID,
		TenantID: tenantID,
		Role:     userRole,
		Locale:   locale,
	}

	// Se existe sessão ativa no MCP, passar para o intent manager
	if state, exists := m.IntentSessions[sessionID]; exists {
		m.Logger.Info("Found active session in MCP",
			"session_id", sessionID,
			"state", state.State,
			"intent", state.PendingIntent.Name)

		// Injetar a sessão no intent manager
		m.IntentManager.InjectSession(sessionID, state)
	}

	// Tentar processar como uma intenção
	if m.IntentManager != nil {
		actionResult, err := m.IntentManager.ProcessMessage(ctx, userMessage, intentCtx)
		// Extrair e salvar o estado da sessão após o processamento
		newState := m.IntentManager.ExtractSession(sessionID)
		if newState != nil {
			m.IntentSessions[sessionID] = newState
			m.Logger.Info("Saved session state",
				"session_id", sessionID,
				"state", newState.State)
		} else {
			// Se não há estado, remover a sessão existente se houver
			if _, exists := m.IntentSessions[sessionID]; exists {
				delete(m.IntentSessions, sessionID)
				m.Logger.Info("Removed completed session", "session_id", sessionID)
			}
		}

		if err == nil && actionResult != nil {
			// Foi processado como intenção, retornar o resultado
			if actionResult.Success {
				// Registrar a mensagem do usuário e a resposta no histórico
				userMsg := Message{
					Role:    "user",
					Content: userMessage,
				}
				systemMsg := Message{
					Role:    "assistant",
					Content: actionResult.Message,
				}

				// Salvar no histórico
				history, exists := m.MessageHistory[sessionID]
				if !exists {
					history = make([]Message, 0)
				}
				history = append(history, userMsg, systemMsg)
				m.MessageHistory[sessionID] = history

				// Retornar a resposta da ação
				return &systemMsg, nil
			} else if actionResult.Message != "" {
				// Foi identificado como intenção, mas houve falha ou está solicitando mais dados
				// Registrar a mensagem do usuário e a resposta no histórico
				userMsg := Message{
					Role:    "user",
					Content: userMessage,
				}
				systemMsg := Message{
					Role:    "assistant",
					Content: actionResult.Message,
				}

				// Salvar no histórico
				history, exists := m.MessageHistory[sessionID]
				if !exists {
					history = make([]Message, 0)
				}
				history = append(history, userMsg, systemMsg)
				m.MessageHistory[sessionID] = history

				// Retornar a resposta da ação
				return &systemMsg, nil
			}
		}
	}

	// Se chegou aqui, não é uma intenção ou não foi possível processá-la
	// Processa como mensagem normal via API do Claude

	// Preparar o sistema prompt com informações contextuais
	systemPrompt := fmt.Sprintf(`Você é Angie, a assistente virtual do Sistema ERP para supermercados.
Você está conversando com o usuário: %s (ID: %s) do tenant: %s com o papel: %s.
Forneça respostas diretas e úteis sobre funcionamento do sistema, relatórios e dados.
Se o usuário quiser realizar ações no sistema como criar usuários, produtos ou clientes, 
você pode processar essas solicitações com comandos específicos.
Para consultas que não exigem ação direta nos dados, forneça informações úteis baseadas
no contexto da conversa.`, userID, userID, tenantID, userRole)

	// Preparar a lista de mensagens para envio
	var messages []Message

	// Adicionar histórico recente
	if history, exists := m.MessageHistory[sessionID]; exists {
		// Adicionar as últimas 10 mensagens ou todas se forem menos que isso
		startIdx := 0
		if len(history) > 10 {
			startIdx = len(history) - 10
		}

		// Filtrar para adicionar somente mensagens do usuário e do sistema, não os systemMessage
		for _, msg := range history[startIdx:] {
			// Pular mensagens do sistema para não confundir o modelo
			if msg.Role != "system" {
				messages = append(messages, msg)
			}
		}
	}

	// Verificar se a mensagem atual do usuário já não está no histórico
	messageFound := false
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content == userMessage {
			messageFound = true
			break
		}
	}

	if !messageFound {
		// Adicionar a mensagem atual do usuário
		messages = append(messages, Message{
			Role:    "user",
			Content: userMessage,
		})
	}

	// Preparar o corpo da requisição
	requestBody := messageRequest{
		Model:     m.Model,
		System:    systemPrompt,
		MaxTokens: 4096,
		Messages:  messages,
	}

	// Serializar para JSON
	reqJSON, err := json.Marshal(requestBody)
	if err != nil {
		m.Logger.Error("Erro ao serializar requisição para Claude API", "error", err)
		return nil, fmt.Errorf("erro ao preparar requisição: %v", err)
	}

	// Log da payload de requisição para debug
	m.Logger.Debug("Payload de requisição para Claude API", "payload", string(reqJSON))

	// Criar a requisição HTTP
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqJSON))
	if err != nil {
		m.Logger.Error("Erro ao criar requisição HTTP para Claude API", "error", err)
		return nil, fmt.Errorf("erro ao criar requisição HTTP: %v", err)
	}

	// Adicionar headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", m.APIKey)
	req.Header.Set("anthropic-beta", "messages-2023-12-15")
	req.Header.Set("Accept", "application/json")

	// Configurar cliente HTTP com timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Enviar requisição
	resp, err := client.Do(req)
	if err != nil {
		m.Logger.Error("Erro ao chamar Claude API", "error", err)
		return nil, fmt.Errorf("erro na comunicação com API: %v", err)
	}
	defer resp.Body.Close()

	// Ler a resposta
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.Logger.Error("Erro ao ler resposta da Claude API", "error", err)
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	// Log da resposta da API para debug
	m.Logger.Debug("Resposta da Claude API", "status", resp.StatusCode, "body", string(respBody))

	// Verificar status da resposta
	if resp.StatusCode != http.StatusOK {
		m.Logger.Error("Erro na resposta da Claude API", "statusCode", resp.StatusCode, "response", string(respBody))
		return nil, fmt.Errorf("erro na API (código %d): %s", resp.StatusCode, string(respBody))
	}

	// Processar a resposta
	var messageResponse messageResponse
	if err := json.Unmarshal(respBody, &messageResponse); err != nil {
		m.Logger.Error("Erro ao decodificar resposta da Claude API", "error", err, "response", string(respBody))
		return nil, fmt.Errorf("erro ao processar resposta: %v", err)
	}

	// Log de informações úteis da resposta
	m.Logger.Info("Resposta processada com sucesso",
		"model", messageResponse.Model,
		"inputTokens", messageResponse.Usage.InputTokens,
		"outputTokens", messageResponse.Usage.OutputTokens,
		"stopReason", messageResponse.StopReason)

	// Criar a mensagem de resposta
	responseMsg := &Message{
		Role:    "assistant",
		Content: messageResponse.Content[0].Text,
	}

	// Salvar a mensagem do usuário e a resposta no histórico
	userMsg := Message{
		Role:    "user",
		Content: userMessage,
	}

	// Obter histórico existente ou inicializar novo
	history, exists := m.MessageHistory[sessionID]
	if !exists {
		history = make([]Message, 0)
	}
	history = append(history, userMsg, *responseMsg)

	// Limitar o tamanho do histórico (opcional, para economizar memória)
	if len(history) > 50 {
		history = history[len(history)-50:]
	}

	m.MessageHistory[sessionID] = history

	return responseMsg, nil
}
