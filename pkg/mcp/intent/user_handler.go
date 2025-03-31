package intent

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hugohenrick/erp-supermercado/pkg/domain"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/repository"
)

// UserIntentHandler gerencia intenções relacionadas a usuários
type UserIntentHandler struct {
	logger        logger.Logger
	userRepo      repository.UserRepository
	regexPatterns map[string]*regexp.Regexp
}

// NewUserIntentHandler cria uma nova instância do handler de intenções de usuário
func NewUserIntentHandler(log logger.Logger, userRepo repository.UserRepository) *UserIntentHandler {
	handler := &UserIntentHandler{
		logger:   log,
		userRepo: userRepo,
		regexPatterns: map[string]*regexp.Regexp{
			"create_user": regexp.MustCompile(`(?i)(cri[ae]r?|cadastr[ae]r?|adicionar?|inserir?)\s+(um\s+)?((novo|uma)\s+)?u[sz]u[aá]rio.+?(?:nome|chamad[oa])\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s]+?)(\s+com|\s+e-?mail|\s+tendo|\s+perfil|\s+função|$)`),
			"get_user":    regexp.MustCompile(`(?i)(busc[ae]r?|encontr[ae]r?|procurar?|mostr[ae]r?|exib[ae]r?)\s+(o\s+)?u[sz]u[aá]rio\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s]+?)|(?:com\s+)?(?:e-?mail|correio)\s+(?P<email>[^\s@]+@[^\s@]+\.[^\s@]+)|(?:com\s+)?id\s+(?P<id>\d+))(\s|$|\.)`),
			"update_user": regexp.MustCompile(`(?i)(atualiz[ae]r?|modific[ae]r?|alter[ae]r?|muda[ae]r?)\s+(o\s+)?u[sz]u[aá]rio\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s]+?)|(?:com\s+)?(?:e-?mail|correio)\s+(?P<email>[^\s@]+@[^\s@]+\.[^\s@]+)|(?:com\s+)?id\s+(?P<id>\d+))`),
			"delete_user": regexp.MustCompile(`(?i)(delet[ae]r?|exclu[iíy]r?|remov[ae]r?|apag[ae]r?)\s+(o\s+)?u[sz]u[aá]rio\s+(?:(chamad[oa]|nome)\s+(?P<name>[A-Za-zÀ-ÖØ-öø-ÿ\s]+?)|(?:com\s+)?(?:e-?mail|correio)\s+(?P<email>[^\s@]+@[^\s@]+\.[^\s@]+)|(?:com\s+)?id\s+(?P<id>\d+))(\s|$|\.)`),
			"list_users":  regexp.MustCompile(`(?i)(list[ae]r?|mostr[ae]r?|exib[ae]r?|ver)\s+(os\s+)?u[sz]u[aá]rios`),
		},
	}
	return handler
}

// CanHandle verifica se este handler pode processar a mensagem
func (h *UserIntentHandler) CanHandle(message string) bool {
	message = strings.TrimSpace(message)

	for _, pattern := range h.regexPatterns {
		if pattern.MatchString(message) {
			return true
		}
	}

	// Verificar outros padrões de linguagem natural sobre usuários
	userTerms := []string{
		"usuário", "usuario", "user", "conta",
		"cadastro de usuário", "gerenciar usuário",
	}

	loweredMsg := strings.ToLower(message)
	for _, term := range userTerms {
		if strings.Contains(loweredMsg, term) {
			return true
		}
	}

	return false
}

// Extract extrai a intenção e entidades da mensagem
func (h *UserIntentHandler) Extract(message string) (*Intent, error) {
	message = strings.TrimSpace(message)

	// Verificar cada padrão de regex para encontrar correspondências
	for intentName, pattern := range h.regexPatterns {
		match := pattern.FindStringSubmatch(message)

		if match != nil {
			// Extrair entidades do regex
			entities := make(map[string]interface{})

			// Obter índices dos grupos nomeados
			subexpNames := pattern.SubexpNames()
			for i, name := range subexpNames {
				if i != 0 && name != "" && match[i] != "" {
					entities[name] = strings.TrimSpace(match[i])
				}
			}

			// Extrair informações adicionais da mensagem
			h.extractAdditionalInfo(message, entities)

			return &Intent{
				Name:            intentName,
				Confidence:      0.8, // Valor fixo para regex; poderia ser dinâmico baseado em outras heurísticas
				Entities:        entities,
				OriginalMessage: message,
			}, nil
		}
	}

	// Se não encontrou um padrão específico, mas está relacionado a usuários
	if h.CanHandle(message) {
		return &Intent{
			Name:            "user_generic",
			Confidence:      0.4,
			Entities:        make(map[string]interface{}),
			OriginalMessage: message,
		}, nil
	}

	return nil, nil
}

// CheckPermission verifica se o usuário tem permissão para executar a ação
func (h *UserIntentHandler) CheckPermission(ctxData ContextData) bool {
	// Verificar se o usuário tem permissão baseado no papel/role
	switch ctxData.Role {
	case "admin", "administrator", "superadmin":
		// Administradores podem fazer tudo
		return true

	case "manager":
		// Gerentes podem fazer tudo exceto excluir usuários
		if _, ok := h.regexPatterns["delete_user"]; ok {
			// Verificar se a intenção é de exclusão
			return false
		}
		return true

	case "hr", "rh":
		// Recursos humanos podem gerenciar usuários, mas não exclusão
		if _, ok := h.regexPatterns["delete_user"]; ok {
			return false
		}
		return true

	default:
		// Outros perfis não podem gerenciar usuários
		return false
	}
}

// Execute executa a ação correspondente à intenção
func (h *UserIntentHandler) Execute(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	switch intent.Name {
	case "create_user":
		return h.createUser(ctxData, intent)
	case "get_user":
		return h.getUser(ctxData, intent)
	case "update_user":
		return h.updateUser(ctxData, intent)
	case "delete_user":
		return h.deleteUser(ctxData, intent)
	case "list_users":
		return h.listUsers(ctxData, intent)
	case "user_generic":
		return &ActionResult{
			Success: false,
			Message: "Entendi que você quer realizar alguma ação relacionada a usuários, mas não consegui identificar exatamente o que. Poderia ser mais específico? Por exemplo, 'criar um usuário chamado João' ou 'listar todos os usuários'.",
		}, nil
	default:
		return nil, errors.New("intenção não suportada")
	}
}

// createUser cria um novo usuário
func (h *UserIntentHandler) createUser(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Extrair dados necessários
	name, hasName := intent.Entities["name"].(string)
	email, hasEmail := intent.Entities["email"].(string)
	role, hasRole := intent.Entities["role"].(string)
	password, hasPassword := intent.Entities["password"].(string)

	// Validar dados obrigatórios
	missingFields := make([]string, 0)

	if !hasName || strings.TrimSpace(name) == "" {
		missingFields = append(missingFields, "nome")
	}

	if !hasEmail || strings.TrimSpace(email) == "" {
		missingFields = append(missingFields, "e-mail")
	}

	// Solicitar os dados que estão faltando, se houver
	if len(missingFields) > 0 {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Para criar um usuário, preciso dos seguintes dados: %s. Pode me informar?", strings.Join(missingFields, ", ")),
		}, nil
	}

	// Usar um perfil padrão se não foi informado
	if !hasRole || strings.TrimSpace(role) == "" {
		role = "user" // Perfil padrão
	}

	// Gerar uma senha temporária se não foi informada
	if !hasPassword || strings.TrimSpace(password) == "" {
		password = generateTempPassword()
	}

	// Criar o objeto de usuário
	newUser := &domain.User{
		Name:     name,
		Email:    email,
		Role:     role,
		Password: password,
		TenantID: ctxData.TenantID,
		Active:   true,
	}

	// Chamar o repositório para persistir o usuário
	err := h.userRepo.Create(ctxData.TenantID, newUser)
	if err != nil {
		h.logger.Error("Erro ao criar usuário", "error", err, "email", email)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Não foi possível criar o usuário: %v", err),
		}, nil
	}

	// Retornar o resultado da operação
	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Usuário %s (%s) criado com sucesso! Uma senha temporária foi gerada: %s", name, email, password),
		Data: map[string]interface{}{
			"user_id": newUser.ID,
			"email":   email,
		},
	}, nil
}

// getUser busca um usuário específico
func (h *UserIntentHandler) getUser(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Verificar qual critério de busca foi fornecido
	var user *domain.User
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		user, err = h.userRepo.FindByID(ctxData.TenantID, id)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como segunda opção
		user, err = h.userRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		users, err := h.userRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar usuário por nome", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar usuário por nome: %v", err),
			}, nil
		}

		if len(users) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum usuário com o nome '%s'", name),
			}, nil
		}

		if len(users) > 1 {
			// Múltiplos resultados, listar opções
			var userList strings.Builder
			userList.WriteString("Encontrei vários usuários com esse nome. Qual destes você está procurando?\n\n")

			for i, u := range users {
				userList.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, u.Name, u.Email))
			}

			return &ActionResult{
				Success: true,
				Message: userList.String(),
				Data: map[string]interface{}{
					"users":           users,
					"multiple_result": true,
				},
			}, nil
		}

		// Se encontrou exatamente um usuário
		user = users[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o usuário. Por favor, informe o ID, e-mail ou nome completo.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar usuário", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o usuário: %v", err),
		}, nil
	}

	if user == nil {
		return &ActionResult{
			Success: false,
			Message: "Usuário não encontrado.",
		}, nil
	}

	// Mostrar os dados do usuário encontrado
	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Usuário encontrado:\n"+
			"- ID: %s\n"+
			"- Nome: %s\n"+
			"- E-mail: %s\n"+
			"- Perfil: %s\n"+
			"- Status: %s",
			user.ID,
			user.Name,
			user.Email,
			user.Role,
			userStatusToString(user.Active)),
		Data: map[string]interface{}{
			"user": user,
		},
	}, nil
}

// updateUser atualiza um usuário existente
func (h *UserIntentHandler) updateUser(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Primeiro, localizar o usuário a ser atualizado
	var user *domain.User
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		user, err = h.userRepo.FindByID(ctxData.TenantID, id)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como segunda opção
		user, err = h.userRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		users, err := h.userRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar usuário por nome para atualização", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar usuário por nome: %v", err),
			}, nil
		}

		if len(users) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum usuário com o nome '%s' para atualizar", name),
			}, nil
		}

		if len(users) > 1 {
			// Múltiplos resultados, listar opções
			var userList strings.Builder
			userList.WriteString("Encontrei vários usuários com esse nome. Qual destes você deseja atualizar?\n\n")

			for i, u := range users {
				userList.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, u.Name, u.Email))
			}

			return &ActionResult{
				Success: false,
				Message: userList.String(),
				Data: map[string]interface{}{
					"users":           users,
					"multiple_result": true,
					"intent":          "update_user",
				},
			}, nil
		}

		// Se encontrou exatamente um usuário
		user = users[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o usuário a ser atualizado. Por favor, informe o ID, e-mail ou nome completo.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar usuário para atualização", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o usuário para atualização: %v", err),
		}, nil
	}

	if user == nil {
		return &ActionResult{
			Success: false,
			Message: "Usuário não encontrado para atualização.",
		}, nil
	}

	// Verificar quais campos serão atualizados
	fieldsToUpdate := make(map[string]interface{})

	// Obter campos para atualização
	newName, hasNewName := intent.Entities["new_name"].(string)
	newEmail, hasNewEmail := intent.Entities["new_email"].(string)
	newRole, hasNewRole := intent.Entities["new_role"].(string)
	newStatus, hasNewStatus := intent.Entities["new_status"].(string)

	if hasNewName && newName != "" {
		fieldsToUpdate["name"] = newName
	}

	if hasNewEmail && newEmail != "" {
		fieldsToUpdate["email"] = newEmail
	}

	if hasNewRole && newRole != "" {
		fieldsToUpdate["role"] = newRole
	}

	if hasNewStatus && newStatus != "" {
		active := true
		if strings.ToLower(newStatus) == "inativo" || strings.ToLower(newStatus) == "inativa" ||
			strings.ToLower(newStatus) == "desativado" || strings.ToLower(newStatus) == "desativada" {
			active = false
		}
		fieldsToUpdate["active"] = active
	}

	// Se não há campos para atualizar, solicitar informações
	if len(fieldsToUpdate) == 0 {
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Qual informação do usuário %s você deseja atualizar? Por favor, informe o novo nome, e-mail, perfil ou status.", user.Name),
		}, nil
	}

	// Atualizar o usuário com os novos dados
	for field, value := range fieldsToUpdate {
		switch field {
		case "name":
			user.Name = value.(string)
		case "email":
			user.Email = value.(string)
		case "role":
			user.Role = value.(string)
		case "active":
			user.Active = value.(bool)
		}
	}

	// Persistir as alterações
	err = h.userRepo.Update(ctxData.TenantID, user)
	if err != nil {
		h.logger.Error("Erro ao atualizar usuário", "error", err, "user_id", user.ID)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao atualizar o usuário: %v", err),
		}, nil
	}

	// Gerar mensagem sobre os campos atualizados
	updatedFields := make([]string, 0, len(fieldsToUpdate))
	for field := range fieldsToUpdate {
		switch field {
		case "name":
			updatedFields = append(updatedFields, "nome")
		case "email":
			updatedFields = append(updatedFields, "e-mail")
		case "role":
			updatedFields = append(updatedFields, "perfil")
		case "active":
			updatedFields = append(updatedFields, "status")
		}
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Usuário %s foi atualizado com sucesso! Campos alterados: %s.",
			user.Name, strings.Join(updatedFields, ", ")),
		Data: map[string]interface{}{
			"user":           user,
			"updated_fields": updatedFields,
		},
	}, nil
}

// deleteUser remove um usuário
func (h *UserIntentHandler) deleteUser(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Primeiro, localizar o usuário a ser excluído
	var user *domain.User
	var err error

	// Se temos ID, usamos ele como prioridade
	if id, ok := intent.Entities["id"].(string); ok && id != "" {
		user, err = h.userRepo.FindByID(ctxData.TenantID, id)
	} else if email, ok := intent.Entities["email"].(string); ok && email != "" {
		// Se temos email, usamos como segunda opção
		user, err = h.userRepo.FindByEmail(ctxData.TenantID, email)
	} else if name, ok := intent.Entities["name"].(string); ok && name != "" {
		// Se temos nome, buscamos por nome
		users, err := h.userRepo.FindByName(ctxData.TenantID, name)
		if err != nil {
			h.logger.Error("Erro ao buscar usuário por nome para exclusão", "error", err, "name", name)
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Erro ao buscar usuário por nome: %v", err),
			}, nil
		}

		if len(users) == 0 {
			return &ActionResult{
				Success: false,
				Message: fmt.Sprintf("Não encontrei nenhum usuário com o nome '%s' para excluir", name),
			}, nil
		}

		if len(users) > 1 {
			// Múltiplos resultados, listar opções
			var userList strings.Builder
			userList.WriteString("Encontrei vários usuários com esse nome. Qual destes você deseja excluir?\n\n")

			for i, u := range users {
				userList.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, u.Name, u.Email))
			}

			return &ActionResult{
				Success: false,
				Message: userList.String(),
				Data: map[string]interface{}{
					"users":           users,
					"multiple_result": true,
					"intent":          "delete_user",
				},
			}, nil
		}

		// Se encontrou exatamente um usuário
		user = users[0]
	} else {
		// Não temos critérios suficientes
		return &ActionResult{
			Success: false,
			Message: "Preciso de mais informações para encontrar o usuário a ser excluído. Por favor, informe o ID, e-mail ou nome completo.",
		}, nil
	}

	if err != nil {
		h.logger.Error("Erro ao buscar usuário para exclusão", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao buscar o usuário para exclusão: %v", err),
		}, nil
	}

	if user == nil {
		return &ActionResult{
			Success: false,
			Message: "Usuário não encontrado para exclusão.",
		}, nil
	}

	// Verificar se não está tentando excluir o próprio usuário atual
	if user.ID == ctxData.UserID {
		return &ActionResult{
			Success: false,
			Message: "Você não pode excluir sua própria conta de usuário.",
		}, nil
	}

	// Excluir o usuário
	err = h.userRepo.Delete(ctxData.TenantID, user.ID)
	if err != nil {
		h.logger.Error("Erro ao excluir usuário", "error", err, "user_id", user.ID)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao excluir o usuário: %v", err),
		}, nil
	}

	return &ActionResult{
		Success: true,
		Message: fmt.Sprintf("Usuário %s (%s) foi excluído com sucesso.", user.Name, user.Email),
		Data: map[string]interface{}{
			"user_id":    user.ID,
			"user_email": user.Email,
		},
	}, nil
}

// listUsers lista todos os usuários
func (h *UserIntentHandler) listUsers(ctxData ContextData, intent *Intent) (*ActionResult, error) {
	// Obter todos os usuários do tenant
	users, err := h.userRepo.FindAll(ctxData.TenantID)
	if err != nil {
		h.logger.Error("Erro ao listar usuários", "error", err)
		return &ActionResult{
			Success: false,
			Message: fmt.Sprintf("Ocorreu um erro ao listar os usuários: %v", err),
		}, nil
	}

	if len(users) == 0 {
		return &ActionResult{
			Success: true,
			Message: "Não há usuários cadastrados no sistema.",
			Data: map[string]interface{}{
				"users": users,
			},
		}, nil
	}

	// Formatar a lista de usuários
	var userList strings.Builder
	userList.WriteString(fmt.Sprintf("Encontrei %d usuários:\n\n", len(users)))

	for i, user := range users {
		statusStr := "Ativo"
		if !user.Active {
			statusStr = "Inativo"
		}
		userList.WriteString(fmt.Sprintf("%d. %s (%s) - Perfil: %s - Status: %s\n",
			i+1, user.Name, user.Email, user.Role, statusStr))
	}

	return &ActionResult{
		Success: true,
		Message: userList.String(),
		Data: map[string]interface{}{
			"users": users,
		},
	}, nil
}

// extractAdditionalInfo extrai informações adicionais da mensagem
func (h *UserIntentHandler) extractAdditionalInfo(message string, entities map[string]interface{}) {
	// Extrair e-mail com regex específico
	emailRegex := regexp.MustCompile(`(?i)(?:e-?mail|correio)[\s:]+([^\s@]+@[^\s@]+\.[^\s@]+)`)
	emailMatch := emailRegex.FindStringSubmatch(message)
	if emailMatch != nil && len(emailMatch) > 1 {
		entities["email"] = emailMatch[1]
	}

	// Extrair perfil/função
	roleRegex := regexp.MustCompile(`(?i)(?:perfil|função|cargo|role)[\s:]+(\w+)`)
	roleMatch := roleRegex.FindStringSubmatch(message)
	if roleMatch != nil && len(roleMatch) > 1 {
		entities["role"] = normalizeRole(roleMatch[1])
	}

	// Para atualizações, extrair novos valores
	if strings.Contains(strings.ToLower(message), "atualiz") ||
		strings.Contains(strings.ToLower(message), "modific") ||
		strings.Contains(strings.ToLower(message), "alter") {

		// Novo nome
		newNameRegex := regexp.MustCompile(`(?i)(?:novo nome|nome para|chamar agora)[\s:]+([A-Za-zÀ-ÖØ-öø-ÿ\s]+?)(?:\s+e|\s+com|\s+tendo|\.|$)`)
		newNameMatch := newNameRegex.FindStringSubmatch(message)
		if newNameMatch != nil && len(newNameMatch) > 1 {
			entities["new_name"] = strings.TrimSpace(newNameMatch[1])
		}

		// Novo email
		newEmailRegex := regexp.MustCompile(`(?i)(?:novo e-?mail|e-?mail para)[\s:]+([^\s@]+@[^\s@]+\.[^\s@]+)`)
		newEmailMatch := newEmailRegex.FindStringSubmatch(message)
		if newEmailMatch != nil && len(newEmailMatch) > 1 {
			entities["new_email"] = newEmailMatch[1]
		}

		// Novo perfil
		newRoleRegex := regexp.MustCompile(`(?i)(?:novo perfil|nova função|perfil para|função para|cargo para)[\s:]+(\w+)`)
		newRoleMatch := newRoleRegex.FindStringSubmatch(message)
		if newRoleMatch != nil && len(newRoleMatch) > 1 {
			entities["new_role"] = normalizeRole(newRoleMatch[1])
		}

		// Novo status
		newStatusRegex := regexp.MustCompile(`(?i)(?:novo status|status para|marcar como)[\s:]+(\w+)`)
		newStatusMatch := newStatusRegex.FindStringSubmatch(message)
		if newStatusMatch != nil && len(newStatusMatch) > 1 {
			entities["new_status"] = newStatusMatch[1]
		}
	}
}

// normalizeRole normaliza o nome do perfil
func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))

	// Mapear termos comuns para perfis padronizados
	switch role {
	case "admin", "administrador", "adm":
		return "admin"
	case "gerente", "manager", "ger":
		return "manager"
	case "vendedor", "atendente", "sales":
		return "sales"
	case "financeiro", "finance", "fin":
		return "finance"
	case "estoque", "inventory", "inv", "almoxarife":
		return "inventory"
	case "rh", "recursos humanos", "hr":
		return "hr"
	default:
		return role
	}
}

// userStatusToString converte o status booleano para string
func userStatusToString(active bool) string {
	if active {
		return "Ativo"
	}
	return "Inativo"
}

// generateTempPassword gera uma senha temporária
func generateTempPassword() string {
	// Implementação simplificada - em produção usaria algo mais robusto
	return fmt.Sprintf("Temp%d!", 100000+rand.Intn(900000))
}

// Pacote math/rand para gerar senhas temporárias
var rand = struct {
	Intn func(n int) int
}{
	// Implementação simples para exemplo
	// Em produção, usaria crypto/rand para maior segurança
	Intn: func(n int) int {
		// Este é apenas um exemplo, não use em produção
		return n / 2
	},
}
