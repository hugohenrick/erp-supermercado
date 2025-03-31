package intent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
)

// IntentManager gerencia o processamento de intenções no MCP
type IntentManager struct {
	// Lista de handlers de intenções registrados
	handlers []IntentHandler

	// Logger para registrar eventos e erros
	logger logger.Logger

	// Armazenamento de estado de conversas ativas
	sessions map[string]*FlowState
}

// NewIntentManager cria uma nova instância do gerenciador de intenções
func NewIntentManager(log logger.Logger) *IntentManager {
	return &IntentManager{
		handlers: make([]IntentHandler, 0),
		logger:   log,
		sessions: make(map[string]*FlowState),
	}
}

// RegisterHandler registra um novo handler de intenção
func (m *IntentManager) RegisterHandler(handler IntentHandler) {
	m.handlers = append(m.handlers, handler)
	m.logger.Info("Handler de intenção registrado", "handler", fmt.Sprintf("%T", handler))
}

// ProcessMessage processa uma mensagem e executa a ação correspondente
func (m *IntentManager) ProcessMessage(ctx context.Context, message string, ctxData ContextData) (*ActionResult, error) {
	// Gerar ID da sessão se não existir
	sessionID := getSessionID(ctxData)

	m.logger.Info("Processing message",
		"session_id", sessionID,
		"message", message,
		"has_active_session", m.sessions[sessionID] != nil,
		"user_id", ctxData.UserID,
		"tenant_id", ctxData.TenantID)

	// Verificar se existe um fluxo em andamento para esta sessão
	if state, exists := m.sessions[sessionID]; exists && state.State != "completed" {
		m.logger.Info("Found active session",
			"session_id", sessionID,
			"state", state.State,
			"intent", state.PendingIntent.Name)

		// Processar mensagem dentro do fluxo existente
		return m.processContinuation(ctx, message, ctxData, sessionID, state)
	}

	m.logger.Info("No active session found, detecting intent", "session_id", sessionID)

	// Detectar intenção na mensagem
	intent, handler, err := m.detectIntent(message)
	if err != nil {
		m.logger.Warn("Failed to detect intent", "error", err)
		return &ActionResult{
			Success: false,
			Message: "Não consegui entender o que você quer fazer. Pode tentar explicar de outra forma?",
		}, nil
	}

	// Se não identificou nenhuma intenção, retornar mensagem genérica
	if intent == nil || handler == nil {
		m.logger.Info("No intent detected in message", "message", message)
		return &ActionResult{
			Success: false,
			Message: "Não identifiquei nenhuma ação específica para executar. Pode ser mais específico?",
		}, nil
	}

	m.logger.Info("Intent detected", "intent", intent.Name, "confidence", intent.Confidence)

	// Verificar permissões
	if !handler.CheckPermission(ctxData) {
		m.logger.Warn("Permission denied", "intent", intent.Name, "role", ctxData.Role)
		return &ActionResult{
			Success: false,
			Message: "Você não tem permissão para executar esta ação. Por favor, contate um administrador se precisar de acesso.",
		}, nil
	}

	// Verificar se é uma operação que requer confirmação
	if requiresConfirmation(intent.Name) {
		// Iniciar fluxo de confirmação
		confirmationState := &FlowState{
			PendingIntent:  intent,
			State:          "awaiting_confirmation",
			CurrentMessage: generateConfirmationMessage(intent),
		}
		m.sessions[sessionID] = confirmationState

		m.logger.Info("Created confirmation flow",
			"session_id", sessionID,
			"intent", intent.Name,
			"state", "awaiting_confirmation")

		return &ActionResult{
			Success: true,
			Message: confirmationState.CurrentMessage,
			Data: map[string]interface{}{
				"awaiting_confirmation": true,
				"intent":                intent.Name,
			},
		}, nil
	}

	// Executar a ação diretamente para operações que não precisam de confirmação
	m.logger.Info("Executing action directly", "intent", intent.Name)
	result, err := handler.Execute(ctxData, intent)
	if err != nil {
		m.logger.Error("Erro ao executar ação", "error", err, "intent", intent.Name)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao executar esta ação: %v", err),
		}, nil
	}

	// Gerar ID de operação para auditoria
	result.OperationID = uuid.New().String()

	return result, nil
}

// processContinuation processa uma mensagem dentro de um fluxo existente
func (m *IntentManager) processContinuation(ctx context.Context, message string, ctxData ContextData, sessionID string, state *FlowState) (*ActionResult, error) {
	// Verificar o estado atual do fluxo
	switch state.State {
	case "awaiting_confirmation":
		// Checar se a mensagem é uma confirmação ou cancelamento
		normalizedMsg := strings.ToLower(strings.TrimSpace(message))

		// Tratamento de confirmação - aceitar muito mais variações
		if normalizedMsg == "confirmar" || normalizedMsg == "sim" || normalizedMsg == "yes" || normalizedMsg == "ok" ||
			normalizedMsg == "confirmado" || normalizedMsg == "confirme" || normalizedMsg == "pode confirmar" ||
			normalizedMsg == "pode cadastrar" || normalizedMsg == "confirmação" || normalizedMsg == "s" ||
			normalizedMsg == "y" || strings.Contains(normalizedMsg, "confirm") ||
			strings.Contains(normalizedMsg, "certo") || strings.Contains(normalizedMsg, "correto") {
			// Encontrar o handler para esta intenção
			var targetHandler IntentHandler
			for _, h := range m.handlers {
				if h.CanHandle(state.PendingIntent.OriginalMessage) {
					targetHandler = h
					break
				}
			}

			if targetHandler == nil {
				// Reset do estado da sessão
				delete(m.sessions, sessionID)
				return &ActionResult{
					Success: false,
					Message: "Desculpe, ocorreu um erro ao processar sua solicitação. Por favor, tente novamente.",
				}, nil
			}

			m.logger.Info("Executando ação confirmada",
				"intent", state.PendingIntent.Name,
				"message", message,
				"user", ctxData.UserID)

			// Executar a ação confirmada
			result, err := targetHandler.Execute(ctxData, state.PendingIntent)

			// Limpar o estado da sessão
			delete(m.sessions, sessionID)

			if err != nil {
				m.logger.Error("Erro ao executar ação confirmada", "error", err, "intent", state.PendingIntent.Name)
				return &ActionResult{
					Success: false,
					Message: fmt.Sprintf("Ocorreu um erro ao executar esta ação: %v", err),
				}, nil
			}

			// Gerar ID de operação para auditoria
			result.OperationID = uuid.New().String()
			return result, nil
		}

		// Tratamento de cancelamento
		if normalizedMsg == "cancelar" || normalizedMsg == "não" || normalizedMsg == "no" || normalizedMsg == "cancel" ||
			normalizedMsg == "n" || strings.Contains(normalizedMsg, "desist") ||
			strings.Contains(normalizedMsg, "canc") {
			// Limpar o estado da sessão
			delete(m.sessions, sessionID)
			return &ActionResult{
				Success: true,
				Message: "Operação cancelada. Posso ajudar com mais alguma coisa?",
			}, nil
		}

		// Se não for confirmação nem cancelamento, pedir novamente
		return &ActionResult{
			Success: false,
			Message: "Por favor, responda 'confirmar' para prosseguir ou 'cancelar' para desistir.",
			Data: map[string]interface{}{
				"awaiting_confirmation": true,
				"intent":                state.PendingIntent.Name,
			},
		}, nil

	case "data_collection":
		// Implementação para coleta de dados adicionais
		// Este é um exemplo simples, mas pode ser expandido conforme necessário
		return &ActionResult{
			Success: false,
			Message: "Precisamos de mais informações para continuar.",
		}, nil

	default:
		// Estado desconhecido, resetar
		delete(m.sessions, sessionID)
		return &ActionResult{
			Success: false,
			Message: "Desculpe, ocorreu um erro. Pode tentar novamente?",
		}, nil
	}
}

// detectIntent tenta identificar uma intenção na mensagem do usuário
func (m *IntentManager) detectIntent(message string) (*Intent, IntentHandler, error) {
	var bestIntent *Intent
	var bestHandler IntentHandler
	var highestConfidence float64 = 0

	// Iterar por todos os handlers registrados
	for _, handler := range m.handlers {
		// Verificar se este handler pode processar a mensagem
		if handler.CanHandle(message) {
			// Extrair intenção e entidades
			intent, err := handler.Extract(message)
			if err != nil {
				m.logger.Error("Erro ao extrair intenção", "error", err, "handler", fmt.Sprintf("%T", handler))
				continue
			}

			// Se encontrou uma intenção com maior confiança, atualizar
			if intent != nil && intent.Confidence > highestConfidence {
				bestIntent = intent
				bestHandler = handler
				highestConfidence = intent.Confidence
			}
		}
	}

	return bestIntent, bestHandler, nil
}

// getSessionID gera um ID de sessão baseado nos dados do contexto
func getSessionID(ctxData ContextData) string {
	return fmt.Sprintf("%s:%s", ctxData.TenantID, ctxData.UserID)
}

// requiresConfirmation determina se uma intenção requer confirmação explícita
func requiresConfirmation(intentName string) bool {
	// Lista de intenções que requerem confirmação
	criticalIntents := []string{
		"create_user",
		"update_user",
		"delete_user",
		"create_product",
		"update_product",
		"delete_product",
		"update_price",
		"create_customer",
		"delete_customer",
	}

	for _, critical := range criticalIntents {
		if intentName == critical {
			return true
		}
	}

	return false
}

// generateConfirmationMessage gera uma mensagem de confirmação baseada na intenção
func generateConfirmationMessage(intent *Intent) string {
	var message string

	switch intent.Name {
	case "create_user":
		name := getStringEntity(intent.Entities, "name")
		email := getStringEntity(intent.Entities, "email")
		role := getStringEntity(intent.Entities, "role")

		message = fmt.Sprintf("Você quer criar um novo usuário com estes dados?\n"+
			"- Nome: %s\n"+
			"- Email: %s\n"+
			"- Perfil: %s\n\n"+
			"Digite 'confirmar' para prosseguir ou 'cancelar' para desistir.",
			name, email, role)

	case "delete_user":
		id := getStringEntity(intent.Entities, "id")
		name := getStringEntity(intent.Entities, "name")

		message = fmt.Sprintf("ATENÇÃO: Você está prestes a EXCLUIR o usuário %s (ID: %s).\n"+
			"Esta ação não pode ser desfeita.\n\n"+
			"Digite 'confirmar' para prosseguir ou 'cancelar' para desistir.",
			name, id)

	case "create_customer":
		name := getStringEntity(intent.Entities, "name")
		document := getStringEntity(intent.Entities, "document")
		email := getStringEntity(intent.Entities, "email")
		phone := getStringEntity(intent.Entities, "phone")
		address := getStringEntity(intent.Entities, "address")

		message = fmt.Sprintf("Confirma a criação do cliente com os seguintes dados?\n"+
			"- Nome: %s\n"+
			"- Documento: %s\n"+
			"- Email: %s\n"+
			"- Telefone: %s\n"+
			"- Endereço: %s\n\n"+
			"Digite 'confirmar' para prosseguir ou 'cancelar' para desistir.",
			name, document, email, phone, address)

	case "delete_customer":
		id := getStringEntity(intent.Entities, "id")
		name := getStringEntity(intent.Entities, "name")

		message = fmt.Sprintf("ATENÇÃO: Você está prestes a EXCLUIR o cliente %s (ID: %s).\n"+
			"Esta ação não pode ser desfeita.\n\n"+
			"Digite 'confirmar' para prosseguir ou 'cancelar' para desistir.",
			name, id)

	// Adicionar outros casos conforme necessário

	default:
		message = "Deseja confirmar esta operação? Digite 'confirmar' para prosseguir ou 'cancelar' para desistir."
	}

	return message
}

// getStringEntity obtém uma entidade como string do mapa de entidades
func getStringEntity(entities map[string]interface{}, key string) string {
	if value, ok := entities[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return "<não informado>"
}

// InjectSession allows an external system to inject a session state
func (m *IntentManager) InjectSession(sessionID string, state *FlowState) {
	m.logger.Info("Injecting session state", "session_id", sessionID, "state", state.State)
	m.sessions[sessionID] = state
}

// ExtractSession returns the current session state for a given session ID
func (m *IntentManager) ExtractSession(sessionID string) *FlowState {
	if state, exists := m.sessions[sessionID]; exists {
		return state
	}
	return nil
}

// GetHandlers returns the list of registered intent handlers
func (m *IntentManager) GetHandlers() []IntentHandler {
	return m.handlers
}
