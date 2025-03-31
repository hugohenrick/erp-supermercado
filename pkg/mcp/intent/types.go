package intent

// Intent representa uma intenção detectada em uma mensagem do usuário
type Intent struct {
	// Nome da intenção (ex: "create_user", "list_products")
	Name string `json:"name"`

	// Confiança na identificação (0-1)
	Confidence float64 `json:"confidence"`

	// Entidades extraídas do texto
	Entities map[string]interface{} `json:"entities"`

	// Mensagem original
	OriginalMessage string `json:"original_message"`
}

// ActionResult representa o resultado de uma ação executada pelo sistema
type ActionResult struct {
	// Sucesso ou falha da operação
	Success bool `json:"success"`

	// Mensagem para o usuário
	Message string `json:"message"`

	// Dados adicionais (depende da ação)
	Data map[string]interface{} `json:"data,omitempty"`

	// ID da operação (para auditoria)
	OperationID string `json:"operation_id,omitempty"`
}

// IntentHandler é a interface para os manipuladores de intenções específicas
type IntentHandler interface {
	// Identifica se esta intenção é aplicável à mensagem
	CanHandle(message string) bool

	// Extrai a intenção e entidades da mensagem
	Extract(message string) (*Intent, error)

	// Executa a ação associada à intenção
	Execute(ctx ContextData, intent *Intent) (*ActionResult, error)

	// Verifica se o usuário tem permissão para executar esta ação
	CheckPermission(ctx ContextData) bool
}

// ContextData contém informações do contexto da requisição
type ContextData struct {
	UserID   string
	TenantID string
	Role     string
	Locale   string
	Session  map[string]interface{}
}

// FlowState rastreia o estado de uma conversa de confirmação
type FlowState struct {
	// Intenção pendente de confirmação
	PendingIntent *Intent `json:"pending_intent,omitempty"`

	// Estado do fluxo (ex: "awaiting_confirmation", "data_collection")
	State string `json:"state"`

	// Dados adicionais para o fluxo
	Data map[string]interface{} `json:"data,omitempty"`

	// Mensagem atual para o usuário
	CurrentMessage string `json:"current_message,omitempty"`
}
