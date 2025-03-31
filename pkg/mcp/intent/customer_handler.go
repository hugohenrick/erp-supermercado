package intent

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/pkg/domain"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// CustomerIntentHandler gerencia intenções relacionadas a clientes
type CustomerIntentHandler struct {
	logger        logger.Logger
	customerRepo  repository.CustomerRepository
	regexPatterns map[string]*regexp.Regexp
}

// NewCustomerIntentHandler cria uma nova instância do handler de intenções de cliente
func NewCustomerIntentHandler(log logger.Logger, customerRepo repository.CustomerRepository) *CustomerIntentHandler {
	handler := &CustomerIntentHandler{
		logger:        log,
		customerRepo:  customerRepo,
		regexPatterns: make(map[string]*regexp.Regexp),
	}

	// Regex patterns to detect different customer-related intents
	// Enhanced patterns with more explicit and variable formats
	handler.regexPatterns["create_customer"] = regexp.MustCompile(
		`(?i)(cadastr[aeo]|cri[ae]r?|adiciona?r?|inserir?|novo|nova|adicionar)\s+(um\s+)?(cliente|pessoa|contato)` +
			`\s*:?\s*(?P<details>[\s\S]*?)(?:\s*\??\s*$|$)`)

	// Add a more structured pattern for form-like inputs that include "Nome:", "CPF:", etc.
	handler.regexPatterns["create_customer_form"] = regexp.MustCompile(
		`(?i)(cadastr[aeo]|cri[ae]r?|adiciona?r?|inserir?|novo|nova)\s+(um\s+)?(cliente|pessoa|contato)\s*:?\s*` +
			`[\r\n]+(Nome\s*:\s*(?P<name>[^\r\n]+))` +
			`(?:[\r\n]+CPF\s*:\s*(?P<document>[^\r\n]+))?` +
			`(?:[\r\n]+Documento\s*:\s*(?P<document2>[^\r\n]+))?` +
			`(?:[\r\n]+CNPJ\s*:\s*(?P<cnpj>[^\r\n]+))?` +
			`(?:[\r\n]+Endereço\s*:\s*(?P<address>[^\r\n]+))?` +
			`(?:[\r\n]+Telefone\s*:\s*(?P<phone>[^\r\n]+))?` +
			`(?:[\r\n]+Email\s*:\s*(?P<email>[^\r\n]+))?` +
			`(?:[\r\n]+Data de Nascimento\s*:\s*(?P<birth_date>[^\r\n]+))?`)

	handler.regexPatterns["get_customer"] = regexp.MustCompile(
		`(?i)(busca[er]|encontr[ea]r|localiza[r]|obter|ver|mostre|pesquis[ae]r?|exiba[r]?)\s+` +
			`(cliente|pessoa|contato)(?:\s+(?:com|por|pelo|pela|de|onde|cujo|id)\s+)?` +
			`(?:(?:id|código|cod|identificador|numero)\s*(?:é|igual a|=|:|\.)\s*(?P<id>[a-zA-Z0-9-]+))?` +
			`(?:(?:email|e-mail)\s*(?:é|igual a|=|:|\.)\s*(?P<email>[^\s,]+))?` +
			`(?:(?:cpf|cnpj|documento)\s*(?:é|igual a|=|:|\.)\s*(?P<document>[0-9.\s\/-]+))?` +
			`(?:(?:nome)\s*(?:é|igual a|=|:|\.)\s*(?P<name>[^?]+?))?` +
			`\s*\??$`)

	handler.regexPatterns["update_customer"] = regexp.MustCompile(
		`(?i)(atualiz[ae]r?|edit[ae]r?|modific[ae]r?|alter[ae]r?)\s+` +
			`(cliente|pessoa|contato)(?:\s+(?:com|por|pelo|pela|de|onde|cujo|id)\s+)?` +
			`(?:(?:id|código|cod|identificador|numero)\s*(?:é|igual a|=|:|\.)\s*(?P<id>[a-zA-Z0-9-]+))?` +
			`(?:(?:email|e-mail)\s*(?:é|igual a|=|:|\.)\s*(?P<email>[^\s,]+))?` +
			`(?:(?:cpf|cnpj|documento)\s*(?:é|igual a|=|:|\.)\s*(?P<document>[0-9.\s\/-]+))?` +
			`(?:(?:nome)\s*(?:é|igual a|=|:|\.)\s*(?P<name>[^,]+?))?` +
			`(?:\s+para|com)?\s*(?P<details>[\s\S]*?)(?:\s*\??\s*$|$)`)

	handler.regexPatterns["delete_customer"] = regexp.MustCompile(
		`(?i)(exclu[ie]r?|delet[ae]r?|apaga[r]?|remov[ae]r?)\s+` +
			`(cliente|pessoa|contato)(?:\s+(?:com|por|pelo|pela|de|onde|cujo|id)\s+)?` +
			`(?:(?:id|código|cod|identificador|numero)\s*(?:é|igual a|=|:|\.)\s*(?P<id>[a-zA-Z0-9-]+))?` +
			`(?:(?:email|e-mail)\s*(?:é|igual a|=|:|\.)\s*(?P<email>[^\s,]+))?` +
			`(?:(?:cpf|cnpj|documento)\s*(?:é|igual a|=|:|\.)\s*(?P<document>[0-9.\s\/-]+))?` +
			`(?:(?:nome)\s*(?:é|igual a|=|:|\.)\s*(?P<name>[^?]+?))?` +
			`\s*\??$`)

	handler.regexPatterns["list_customers"] = regexp.MustCompile(
		`(?i)(lista[r]?|mostrar?|exib[ae]r?|ver?|recuper[ae]r?)\s+(todo[s]?|o[s]?|a[s]?)?\s*` +
			`(clientes|pessoas|contatos)\s*\??$`)

	return handler
}

// CanHandle verifica se este handler pode processar a mensagem
func (h *CustomerIntentHandler) CanHandle(message string) bool {
	if message == "" {
		return false
	}

	// Add debug logging to see if the handler is being considered
	h.logger.Debug("Checking if customer handler can handle message", "message", message)

	// Check each pattern
	for intent, pattern := range h.regexPatterns {
		if pattern.MatchString(message) {
			h.logger.Info("Message matches customer intent pattern", "intent", intent)

			// Add special logging for form-style messages
			if intent == "create_customer_form" {
				h.logger.Info("Message contains form-style customer creation")
			}

			return true
		}
	}

	return false
}

// Extract extrai a intenção e entidades da mensagem
func (h *CustomerIntentHandler) Extract(message string) (*Intent, error) {
	// Add debug logging
	h.logger.Debug("Extracting intent from message", "message", message)

	// FORCE MATCHING for exact pattern we're getting in logs
	if strings.Contains(message, "Nome:") &&
		strings.Contains(message, "CPF:") &&
		strings.Contains(message, "Endereço:") {

		h.logger.Info("FORCE MATCHING exact customer creation pattern")

		entities := make(map[string]interface{})

		// Extract each field with basic line patterns
		nameRegex := regexp.MustCompile(`(?im)^Nome\s*:\s*(.+)$`)
		if matches := nameRegex.FindStringSubmatch(message); len(matches) > 1 {
			entities["name"] = strings.TrimSpace(matches[1])
		}

		docRegex := regexp.MustCompile(`(?im)^CPF\s*:\s*(.+)$`)
		if matches := docRegex.FindStringSubmatch(message); len(matches) > 1 {
			entities["document"] = strings.TrimSpace(matches[1])
		}

		addrRegex := regexp.MustCompile(`(?im)^Endereço\s*:\s*(.+)$`)
		if matches := addrRegex.FindStringSubmatch(message); len(matches) > 1 {
			entities["address"] = strings.TrimSpace(matches[1])
		}

		phoneRegex := regexp.MustCompile(`(?im)^Telefone\s*:\s*(.+)$`)
		if matches := phoneRegex.FindStringSubmatch(message); len(matches) > 1 {
			entities["phone"] = strings.TrimSpace(matches[1])
		}

		emailRegex := regexp.MustCompile(`(?im)^Email\s*:\s*(.+)$`)
		if matches := emailRegex.FindStringSubmatch(message); len(matches) > 1 {
			entities["email"] = strings.TrimSpace(matches[1])
		}

		h.logger.Info("Force extracted entities",
			"name", entities["name"],
			"document", entities["document"],
			"address", entities["address"],
			"phone", entities["phone"],
			"email", entities["email"])

		return &Intent{
			Name:            "create_customer",
			OriginalMessage: message,
			Entities:        entities,
			Confidence:      0.99,
		}, nil
	}

	// First check for create_customer_form as it's more specific
	if matches := h.regexPatterns["create_customer_form"].FindStringSubmatch(message); matches != nil {
		h.logger.Info("Detected structured customer creation message")

		// Get named captures
		names := h.regexPatterns["create_customer_form"].SubexpNames()
		entities := make(map[string]interface{})

		for i, name := range names {
			if i != 0 && name != "" && i < len(matches) && matches[i] != "" {
				entities[name] = strings.TrimSpace(matches[i])
				h.logger.Debug("Extracted customer entity", "name", name, "value", entities[name])
			}
		}

		// Combine document fields if needed
		if entities["document"] == nil && entities["document2"] != nil {
			entities["document"] = entities["document2"]
			delete(entities, "document2")
		}
		if entities["document"] == nil && entities["cnpj"] != nil {
			entities["document"] = entities["cnpj"]
			delete(entities, "cnpj")
		}

		return &Intent{
			Name:            "create_customer",
			OriginalMessage: message,
			Entities:        entities,
			Confidence:      0.9,
		}, nil
	}

	// Special case for confirmations
	loweredMsg := strings.ToLower(message)
	if len(message) < 20 && (strings.Contains(loweredMsg, "sim") ||
		strings.Contains(loweredMsg, "confirma") ||
		strings.Contains(loweredMsg, "confirmado") ||
		strings.Contains(loweredMsg, "ok") ||
		strings.Contains(loweredMsg, "correto") ||
		strings.Contains(loweredMsg, "pode cadastrar")) {

		h.logger.Info("Detected confirmation message", "message", message)
		return &Intent{
			Name:            "confirm_action",
			Confidence:      0.9,
			Entities:        make(map[string]interface{}),
			OriginalMessage: message,
		}, nil
	}

	// Verificar cada padrão de regex para encontrar correspondências
	for intentName, pattern := range h.regexPatterns {
		match := pattern.FindStringSubmatch(message)

		if match != nil {
			h.logger.Info("Matched regex pattern", "pattern", intentName)

			// Extrair entidades do regex
			entities := make(map[string]interface{})

			// Obter índices dos grupos nomeados
			subexpNames := pattern.SubexpNames()
			for i, name := range subexpNames {
				if i != 0 && name != "" && match[i] != "" {
					entities[name] = strings.TrimSpace(match[i])
					h.logger.Info("Extracted entity", "name", name, "value", match[i])
				}
			}

			// For form-style inputs, handle the special case
			if intentName == "create_customer_form" && len(match) > 6 {
				entities["name"] = strings.TrimSpace(match[6])
				entities["document"] = strings.TrimSpace(match[8])
				h.logger.Info("Extracted form entities",
					"name", entities["name"],
					"document", entities["document"])
			}

			// Extrair informações adicionais da mensagem
			h.extractAdditionalInfo(message, entities)

			// Mapear 'n' para 'name' se existir
			if n, ok := entities["n"]; ok {
				entities["name"] = n
				delete(entities, "n")
			}

			// Mapear 'doc' para 'document' se existir
			if doc, ok := entities["doc"]; ok {
				entities["document"] = doc
				delete(entities, "doc")
			}

			// If this is a create_customer_form, change the intent to create_customer
			if intentName == "create_customer_form" {
				intentName = "create_customer"
			}

			return &Intent{
				Name:            intentName,
				Confidence:      0.8, // Valor fixo para regex
				Entities:        entities,
				OriginalMessage: message,
			}, nil
		}
	}

	// If we have a structured message with Name and CPF/CNPJ, assume customer creation
	if strings.Contains(message, "Nome:") && (strings.Contains(message, "CPF:") || strings.Contains(message, "CNPJ:") || strings.Contains(message, "Documento:")) {
		h.logger.Info("Detected structured customer creation message")
		entities := make(map[string]interface{})

		// Extract name
		nameRegex := regexp.MustCompile(`(?i)Nome[\s:]+(.*?)(\n|$)`)
		nameMatch := nameRegex.FindStringSubmatch(message)
		if nameMatch != nil && len(nameMatch) > 1 {
			entities["name"] = strings.TrimSpace(nameMatch[1])
		}

		// Extract document
		docRegex := regexp.MustCompile(`(?i)(CPF|CNPJ|Documento)[\s:]+(.*?)(\n|$)`)
		docMatch := docRegex.FindStringSubmatch(message)
		if docMatch != nil && len(docMatch) > 2 {
			entities["document"] = strings.TrimSpace(docMatch[2])
		}

		h.extractAdditionalInfo(message, entities)

		return &Intent{
			Name:            "create_customer",
			Confidence:      0.7,
			Entities:        entities,
			OriginalMessage: message,
		}, nil
	}

	// Se não encontrou um padrão específico, mas está relacionado a clientes
	if h.CanHandle(message) {
		h.logger.Info("No specific intent pattern matched, but can handle as generic customer message")
		return &Intent{
			Name:            "customer_generic",
			Confidence:      0.4,
			Entities:        make(map[string]interface{}),
			OriginalMessage: message,
		}, nil
	}

	h.logger.Info("No intent detected")
	return nil, nil
}

// CheckPermission verifica se o usuário tem permissão para executar a ação
func (h *CustomerIntentHandler) CheckPermission(ctx ContextData) bool {
	// Verificar se o usuário tem permissão baseado no papel
	// Admin e manager podem gerenciar clientes
	if ctx.Role == "admin" || ctx.Role == "manager" || ctx.Role == "sales" {
		return true
	}
	return false
}

// Execute executa a ação associada à intenção
func (h *CustomerIntentHandler) Execute(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	h.logger.Info("Executing intent", "intent", intent.Name, "entities", fmt.Sprintf("%v", intent.Entities))

	switch intent.Name {
	case "create_customer":
		return h.createCustomer(ctxData, intent)
	case "get_customer":
		return h.getCustomer(ctxData, intent)
	case "update_customer":
		return h.updateCustomer(ctxData, intent)
	case "delete_customer":
		return h.deleteCustomer(ctxData, intent)
	case "list_customers":
		return h.listCustomers(ctxData, intent)
	case "confirm_action":
		// For a standalone confirmation, just acknowledge it
		return &ActionResult{
			Success: true,
			Message: "Ação confirmada. O que você gostaria de fazer agora?",
		}, nil
	case "customer_generic":
		return &ActionResult{
			Success: false,
			Message: "Entendi que você quer realizar alguma ação relacionada a clientes, mas não consegui identificar exatamente o que. Poderia ser mais específico? Por exemplo, 'criar um cliente chamado João Silva' ou 'listar todos os clientes'.",
		}, nil
	default:
		return nil, errors.New("intenção não suportada")
	}
}

// createCustomer cria um novo cliente
func (h *CustomerIntentHandler) createCustomer(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Verificar se temos o nome do cliente
	name, ok := intent.Entities["name"].(string)
	if !ok || name == "" {
		h.logger.Error("Nome do cliente não fornecido",
			"entities", fmt.Sprintf("%v", intent.Entities))

		// Deep inspect all entities for debugging
		entityDebug := ""
		for k, v := range intent.Entities {
			entityDebug += fmt.Sprintf("%s=%v, ", k, v)
		}

		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Para cadastrar um cliente, preciso pelo menos do nome. Por favor, informe o nome completo. (Debug: entidades extraídas: %s)", entityDebug),
		}, nil
	}

	h.logger.Info("Iniciando criação de cliente",
		"name", name,
		"tenant_id", ctxData.TenantID,
		"role", ctxData.Role)

	// Verificar se já existe um cliente com este nome ou documento
	if document, ok := intent.Entities["document"].(string); ok && document != "" {
		existingCustomer, err := h.customerRepo.FindByDocument(ctxData.TenantID, document)
		if err == nil && existingCustomer != nil {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Já existe um cliente com o documento '%s': %s.", document, existingCustomer.Name),
			}, nil
		}
	}

	if email, ok := intent.Entities["email"].(string); ok && email != "" {
		existingCustomer, err := h.customerRepo.FindByEmail(ctxData.TenantID, email)
		if err == nil && existingCustomer != nil {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Já existe um cliente com o email '%s': %s.", email, existingCustomer.Name),
			}, nil
		}
	}

	// Verificar se já existe um cliente com este nome
	customers, err := h.customerRepo.FindByName(ctxData.TenantID, name)
	if err != nil {
		h.logger.Error("Erro ao verificar se cliente já existe", "error", err, "name", name)
	} else if len(customers) > 0 {
		// Existe cliente com o mesmo nome
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Já existe um cliente chamado '%s'. Deseja atualizá-lo ou criar um novo cliente com outro nome?", name),
		}, nil
	}

	// Criar o novo cliente
	customer := &domain.Customer{
		ID:           uuid.New().String(),
		Name:         name,
		TenantID:     ctxData.TenantID,
		Active:       true,
		CustomerType: "PF", // Valor padrão
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Preencher campos adicionais se informados
	if email, ok := intent.Entities["email"].(string); ok && email != "" {
		customer.Email = email
	}

	if document, ok := intent.Entities["document"].(string); ok && document != "" {
		customer.Document = document
		// Determinar tipo de cliente com base no documento (CPF ou CNPJ)
		if len(document) > 11 {
			customer.CustomerType = "PJ"
		} else {
			customer.CustomerType = "PF"
		}
	}

	if phone, ok := intent.Entities["phone"].(string); ok && phone != "" {
		customer.Phone = phone
	}

	if address, ok := intent.Entities["address"].(string); ok && address != "" {
		customer.Address = address
	}

	if city, ok := intent.Entities["city"].(string); ok && city != "" {
		customer.City = city
	}

	if state, ok := intent.Entities["state"].(string); ok && state != "" {
		customer.State = state
	}

	if zipCode, ok := intent.Entities["zip_code"].(string); ok && zipCode != "" {
		customer.ZipCode = zipCode
	}

	// Depurar os dados do cliente antes de salvar
	h.logger.Info("Tentando salvar cliente",
		"id", customer.ID,
		"name", customer.Name,
		"email", customer.Email,
		"document", customer.Document,
		"phone", customer.Phone,
		"address", customer.Address,
		"tenant_id", customer.TenantID)

	// Salvar o cliente no banco de dados
	err = h.customerRepo.Create(ctxData.TenantID, customer)
	if err != nil {
		h.logger.Error("Erro ao criar cliente",
			"error", err,
			"customer", name,
			"repo_type", fmt.Sprintf("%T", h.customerRepo))

		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Não foi possível criar o cliente: %v (erro técnico: %T)", err, err),
			Data: map[string]interface{}{
				"error":     err.Error(),
				"repo_type": fmt.Sprintf("%T", h.customerRepo),
			},
		}, nil
	}

	// Retornar o resultado da operação
	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("✅ Cliente '%s' cadastrado com sucesso! O ID do novo cliente é #%s.", name, customer.ID),
		Data: map[string]interface{}{
			"customer_id": customer.ID,
		},
	}, nil
}

// getCustomer busca um cliente específico
func (h *CustomerIntentHandler) getCustomer(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Verificar qual critério de busca foi fornecido
	var customer *domain.Customer
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		customer, err = h.customerRepo.FindByID(ctxData.TenantID, id)
	} else if document, ok := intent.Entities["document"].(string); ok && document != "" {
		// Se temos documento, usamos como segunda opção
		customer, err = h.customerRepo.FindByDocument(ctxData.TenantID, document)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como terceira opção
		customer, err = h.customerRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		customers, err := h.customerRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar cliente por nome", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar cliente por nome: %v", err),
			}, nil
		}

		if len(customers) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum cliente com o nome '%s'", name),
			}, nil
		}

		if len(customers) > 1 {
			// Múltiplos resultados, listar opções
			var customerList strings.Builder
			customerList.WriteString("Encontrei vários clientes com esse nome. Qual destes você está procurando?\n\n")

			for i, c := range customers {
				customerList.WriteString(fmt.Sprintf("%d. %s", i+1, c.Name))
				if c.Document != "" {
					customerList.WriteString(fmt.Sprintf(" (Documento: %s)", c.Document))
				}
				if c.Email != "" {
					customerList.WriteString(fmt.Sprintf(" (Email: %s)", c.Email))
				}
				customerList.WriteString("\n")
			}

			return &ActionResult{
				Success: true,
				Message: customerList.String(),
				Data: map[string]interface{}{
					"customers":       customers,
					"multiple_result": true,
				},
			}, nil
		}

		// Se encontrou exatamente um cliente
		customer = customers[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o cliente. Por favor, informe o ID, documento, e-mail ou nome.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar cliente", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o cliente: %v", err),
		}, nil
	}

	if customer == nil {
		return &ActionResult{
			Success: false,
			Message: "Cliente não encontrado.",
		}, nil
	}

	// Mostrar os dados do cliente encontrado
	customerType := "Pessoa Física"
	if customer.CustomerType == "PJ" {
		customerType = "Pessoa Jurídica"
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Cliente encontrado:\n"+
			"- ID: %s\n"+
			"- Nome: %s\n"+
			"- Tipo: %s\n"+
			"- Documento: %s\n"+
			"- Email: %s\n"+
			"- Telefone: %s\n"+
			"- Endereço: %s, %s - %s\n"+
			"- CEP: %s\n"+
			"- Status: %s",
			customer.ID,
			customer.Name,
			customerType,
			customer.Document,
			customer.Email,
			customer.Phone,
			customer.Address,
			customer.City,
			customer.State,
			customer.ZipCode,
			customerStatusToString(customer.Active)),
		Data: map[string]interface{}{
			"customer": customer,
		},
	}, nil
}

// updateCustomer atualiza os dados de um cliente
func (h *CustomerIntentHandler) updateCustomer(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Primeiro, localizar o cliente a ser atualizado
	var customer *domain.Customer
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		customer, err = h.customerRepo.FindByID(ctxData.TenantID, id)
	} else if document, ok := intent.Entities["document"].(string); ok && document != "" {
		// Se temos documento, usamos como segunda opção
		customer, err = h.customerRepo.FindByDocument(ctxData.TenantID, document)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como terceira opção
		customer, err = h.customerRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		customers, err := h.customerRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar cliente por nome para atualização", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar cliente por nome: %v", err),
			}, nil
		}

		if len(customers) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum cliente com o nome '%s' para atualizar", name),
			}, nil
		}

		if len(customers) > 1 {
			// Múltiplos resultados, listar opções
			var customerList strings.Builder
			customerList.WriteString("Encontrei vários clientes com esse nome. Qual destes você deseja atualizar?\n\n")

			for i, c := range customers {
				customerList.WriteString(fmt.Sprintf("%d. %s", i+1, c.Name))
				if c.Document != "" {
					customerList.WriteString(fmt.Sprintf(" (Documento: %s)", c.Document))
				}
				if c.Email != "" {
					customerList.WriteString(fmt.Sprintf(" (Email: %s)", c.Email))
				}
				customerList.WriteString("\n")
			}

			return &ActionResult{
				Success: false,
				Message: customerList.String(),
				Data: map[string]interface{}{
					"customers":       customers,
					"multiple_result": true,
					"intent":          "update_customer",
				},
			}, nil
		}

		// Se encontrou exatamente um cliente
		customer = customers[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o cliente a ser atualizado. Por favor, informe o ID, documento, e-mail ou nome.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar cliente para atualização", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o cliente: %v", err),
		}, nil
	}

	if customer == nil {
		return &ActionResult{
			Success: false,
			Message: "Cliente não encontrado.",
		}, nil
	}

	// Atualizar os campos que foram informados
	updated := false

	if newName, ok := intent.Entities["new_name"].(string); ok && newName != "" && newName != customer.Name {
		customer.Name = newName
		updated = true
	}

	if newEmail, ok := intent.Entities["new_email"].(string); ok && newEmail != "" && newEmail != customer.Email {
		customer.Email = newEmail
		updated = true
	}

	if newDocument, ok := intent.Entities["new_document"].(string); ok && newDocument != "" && newDocument != customer.Document {
		customer.Document = newDocument
		// Atualizar tipo de cliente com base no documento
		if len(newDocument) > 11 {
			customer.CustomerType = "PJ"
		} else {
			customer.CustomerType = "PF"
		}
		updated = true
	}

	if newPhone, ok := intent.Entities["phone"].(string); ok && newPhone != "" && newPhone != customer.Phone {
		customer.Phone = newPhone
		updated = true
	}

	if newAddress, ok := intent.Entities["address"].(string); ok && newAddress != "" && newAddress != customer.Address {
		customer.Address = newAddress
		updated = true
	}

	if newCity, ok := intent.Entities["city"].(string); ok && newCity != "" && newCity != customer.City {
		customer.City = newCity
		updated = true
	}

	if newState, ok := intent.Entities["state"].(string); ok && newState != "" && newState != customer.State {
		customer.State = newState
		updated = true
	}

	if newZipCode, ok := intent.Entities["zip_code"].(string); ok && newZipCode != "" && newZipCode != customer.ZipCode {
		customer.ZipCode = newZipCode
		updated = true
	}

	if active, ok := intent.Entities["active"].(string); ok && active != "" {
		if strings.ToLower(active) == "ativo" || strings.ToLower(active) == "sim" || strings.ToLower(active) == "true" {
			if !customer.Active {
				customer.Active = true
				updated = true
			}
		} else if strings.ToLower(active) == "inativo" || strings.ToLower(active) == "não" || strings.ToLower(active) == "nao" || strings.ToLower(active) == "false" {
			if customer.Active {
				customer.Active = false
				updated = true
			}
		}
	}

	if !updated {
		return &ActionResult{
			Success: false,
			Message: "Nenhuma informação nova foi fornecida para atualizar o cliente. Por favor, informe quais dados deseja atualizar.",
		}, nil
	}

	// Salvar as atualizações
	err = h.customerRepo.Update(ctxData.TenantID, customer)
	if err != nil {
		h.logger.Error("Erro ao atualizar cliente", "error", err, "customer_id", customer.ID)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Não foi possível atualizar o cliente: %v", err),
		}, nil
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Cliente '%s' atualizado com sucesso!", customer.Name),
		Data: map[string]interface{}{
			"customer": customer,
		},
	}, nil
}

// deleteCustomer remove um cliente
func (h *CustomerIntentHandler) deleteCustomer(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Primeiro, localizar o cliente a ser excluído
	var customer *domain.Customer
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		customer, err = h.customerRepo.FindByID(ctxData.TenantID, id)
	} else if document, ok := intent.Entities["document"].(string); ok && document != "" {
		// Se temos documento, usamos como segunda opção
		customer, err = h.customerRepo.FindByDocument(ctxData.TenantID, document)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como terceira opção
		customer, err = h.customerRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		customers, err := h.customerRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar cliente por nome para exclusão", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar cliente por nome: %v", err),
			}, nil
		}

		if len(customers) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum cliente com o nome '%s' para excluir", name),
			}, nil
		}

		if len(customers) > 1 {
			// Múltiplos resultados, listar opções
			var customerList strings.Builder
			customerList.WriteString("Encontrei vários clientes com esse nome. Qual destes você deseja excluir?\n\n")

			for i, c := range customers {
				customerList.WriteString(fmt.Sprintf("%d. %s", i+1, c.Name))
				if c.Document != "" {
					customerList.WriteString(fmt.Sprintf(" (Documento: %s)", c.Document))
				}
				if c.Email != "" {
					customerList.WriteString(fmt.Sprintf(" (Email: %s)", c.Email))
				}
				customerList.WriteString("\n")
			}

			return &ActionResult{
				Success: false,
				Message: customerList.String(),
				Data: map[string]interface{}{
					"customers":       customers,
					"multiple_result": true,
					"intent":          "delete_customer",
				},
			}, nil
		}

		// Se encontrou exatamente um cliente
		customer = customers[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o cliente a ser excluído. Por favor, informe o ID, documento, e-mail ou nome completo.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar cliente para exclusão", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o cliente: %v", err),
		}, nil
	}

	if customer == nil {
		return &ActionResult{
			Success: false,
			Message: "Cliente não encontrado.",
		}, nil
	}

	// Excluir o cliente
	err = h.customerRepo.Delete(ctxData.TenantID, customer.ID)
	if err != nil {
		h.logger.Error("Erro ao excluir cliente", "error", err, "customer_id", customer.ID)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Não foi possível excluir o cliente: %v", err),
		}, nil
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Cliente '%s' excluído com sucesso!", customer.Name),
	}, nil
}

// listCustomers lista os clientes
func (h *CustomerIntentHandler) listCustomers(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Implementação básica: listar os primeiros 10 clientes
	customers, err := h.customerRepo.FindAll(ctxData.TenantID)
	if err != nil {
		h.logger.Error("Erro ao listar clientes", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao listar os clientes: %v", err),
		}, nil
	}

	if len(customers) == 0 {
		return &ActionResult{
			Success: true,
			Message: "Não há clientes cadastrados.",
		}, nil
	}

	// Criar a mensagem de resposta
	var customerList strings.Builder
	customerList.WriteString(fmt.Sprintf("Encontrei %d clientes cadastrados. Aqui estão os primeiros 10:\n\n", len(customers)))

	displayCount := len(customers)
	if displayCount > 10 {
		displayCount = 10
	}

	for i := 0; i < displayCount; i++ {
		c := customers[i]
		customerList.WriteString(fmt.Sprintf("%d. %s", i+1, c.Name))
		if c.Document != "" {
			customerList.WriteString(fmt.Sprintf(" (Documento: %s)", c.Document))
		}
		if c.Email != "" {
			customerList.WriteString(fmt.Sprintf(" (Email: %s)", c.Email))
		}
		customerList.WriteString("\n")
	}

	if len(customers) > 10 {
		customerList.WriteString(fmt.Sprintf("\nExistem mais %d clientes. Para ver mais detalhes, especifique um cliente pelo nome, documento ou ID.", len(customers)-10))
	}

	return &ActionResult{
		Success: true,
		Message: customerList.String(),
		Data: map[string]interface{}{
			"total_customers": len(customers),
			"customers":       customers[:displayCount],
		},
	}, nil
}

// extractAdditionalInfo extrai informações adicionais da mensagem
func (h *CustomerIntentHandler) extractAdditionalInfo(message string, entities map[string]interface{}) {
	// Extrair email com regex específico
	emailRegex := regexp.MustCompile(`(?i)(?:e-?mail|correio)[\s:]+([^\s@]+@[^\s@]+\.[^\s@]+)`)
	emailMatch := emailRegex.FindStringSubmatch(message)
	if emailMatch != nil && len(emailMatch) > 1 {
		entities["email"] = emailMatch[1]
	}

	// Extrair documento (CPF/CNPJ)
	documentRegex := regexp.MustCompile(`(?i)(?:documento|cpf|cnpj)[\s:]+([0-9.-/]+)`)
	documentMatch := documentRegex.FindStringSubmatch(message)
	if documentMatch != nil && len(documentMatch) > 1 {
		entities["document"] = documentMatch[1]
	}

	// Extrair telefone
	phoneRegex := regexp.MustCompile(`(?i)(?:telefone|fone|celular)[\s:]+([0-9()+ -]+)`)
	phoneMatch := phoneRegex.FindStringSubmatch(message)
	if phoneMatch != nil && len(phoneMatch) > 1 {
		entities["phone"] = phoneMatch[1]
	}

	// Extrair endereço
	addressRegex := regexp.MustCompile(`(?i)(?:endereço|endereco)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s0-9,.º-]+?)(?:\s+em|\s+na cidade|\s+no|\s+tendo|\.|$)`)
	addressMatch := addressRegex.FindStringSubmatch(message)
	if addressMatch != nil && len(addressMatch) > 1 {
		entities["address"] = strings.TrimSpace(addressMatch[1])
	}

	// Extrair cidade
	cityRegex := regexp.MustCompile(`(?i)(?:cidade|localidade)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s-]+?)(?:\s+e|\s+no|\s+em|\s+tendo|\.|$)`)
	cityMatch := cityRegex.FindStringSubmatch(message)
	if cityMatch != nil && len(cityMatch) > 1 {
		entities["city"] = strings.TrimSpace(cityMatch[1])
	}

	// Extrair estado
	stateRegex := regexp.MustCompile(`(?i)(?:estado|UF)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ]{2,20})`)
	stateMatch := stateRegex.FindStringSubmatch(message)
	if stateMatch != nil && len(stateMatch) > 1 {
		entities["state"] = strings.TrimSpace(stateMatch[1])
	}

	// Extrair CEP
	zipCodeRegex := regexp.MustCompile(`(?i)(?:cep|código postal)[\s:]+([0-9]{5}[-]?[0-9]{3})`)
	zipCodeMatch := zipCodeRegex.FindStringSubmatch(message)
	if zipCodeMatch != nil && len(zipCodeMatch) > 1 {
		entities["zip_code"] = zipCodeMatch[1]
	}

	// Para atualizações, extrair novos valores
	// Verificar se é uma atualização pelo contexto da mensagem
	if strings.Contains(strings.ToLower(message), "atualiz") ||
		strings.Contains(strings.ToLower(message), "alter") ||
		strings.Contains(strings.ToLower(message), "mud") {

		// Extrair novo nome
		newNameRegex := regexp.MustCompile(`(?i)(?:novo nome|mudar nome para|atualizar nome para|alterar nome para)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s]+?)(?:\s+e|\s+com|\s+tendo|\.|$)`)
		newNameMatch := newNameRegex.FindStringSubmatch(message)
		if newNameMatch != nil && len(newNameMatch) > 1 {
			entities["new_name"] = strings.TrimSpace(newNameMatch[1])
		}

		// Extrair novo email
		newEmailRegex := regexp.MustCompile(`(?i)(?:novo e-?mail|mudar e-?mail para|atualizar e-?mail para|alterar e-?mail para)[\s:]+([^\s@]+@[^\s@]+\.[^\s@]+)`)
		newEmailMatch := newEmailRegex.FindStringSubmatch(message)
		if newEmailMatch != nil && len(newEmailMatch) > 1 {
			entities["new_email"] = newEmailMatch[1]
		}

		// Extrair novo documento
		newDocRegex := regexp.MustCompile(`(?i)(?:novo documento|novo cpf|novo cnpj|mudar documento para|atualizar documento para)[\s:]+([0-9.-/]+)`)
		newDocMatch := newDocRegex.FindStringSubmatch(message)
		if newDocMatch != nil && len(newDocMatch) > 1 {
			entities["new_document"] = newDocMatch[1]
		}
	}
}

// customerStatusToString converte o status booleano para string legível
func customerStatusToString(active bool) string {
	if active {
		return "Ativo"
	}
	return "Inativo"
}
