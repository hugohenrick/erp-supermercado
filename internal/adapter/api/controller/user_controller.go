package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/domain/user"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// UserController gerencia as requisições relacionadas a usuários
type UserController struct {
	userRepository user.Repository
}

// NewUserController cria uma nova instância de UserController
func NewUserController(userRepository user.Repository) *UserController {
	return &UserController{
		userRepository: userRepository,
	}
}

// Create cria um novo usuário
// @Summary Cria um novo usuário
// @Description Cria um novo usuário no sistema
// @Tags users
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param user body dto.UserRequest true "Dados do usuário"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users [post]
func (c *UserController) Create(ctx *gin.Context) {
	var request dto.UserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Verificar se a senha foi fornecida (obrigatória para novos usuários)
	if request.Password == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Senha requerida", "A senha é obrigatória para novos usuários"))
		return
	}

	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	// Gerar um ID para o novo usuário
	id := uuid.New().String()

	// Criar o modelo de domínio a partir do DTO
	u := &user.User{
		ID:          id,
		TenantID:    tenantID,
		BranchID:    request.BranchID,
		Name:        request.Name,
		Email:       request.Email,
		Role:        user.Role(request.Role),
		Status:      user.StatusActive, // Por padrão, o usuário é criado ativo
		LastLoginAt: time.Time{},       // Zero value para data de último login
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Definir a senha com hash
	if err := u.SetPassword(request.Password); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao processar senha", err.Error()))
		return
	}

	// Persistir o usuário
	err := c.userRepository.Create(ctx, u)
	if err != nil {
		if errors.Is(err, repository.ErrUserDuplicateEmail) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Usuário com mesmo email já existe", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao criar usuário", err.Error()))
		return
	}

	// Retornar o usuário criado
	ctx.JSON(http.StatusCreated, dto.ToUserResponse(u))
}

// GetByID busca um usuário pelo ID
// @Summary Busca um usuário pelo ID
// @Description Busca um usuário pelo seu ID
// @Tags users
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID do usuário"
// @Success 200 {object} dto.UserResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{id} [get]
func (c *UserController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	u, err := c.userRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToUserResponse(u))
}

// List lista os usuários com paginação
// @Summary Lista os usuários
// @Description Lista os usuários do tenant com paginação
// @Tags users
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param page query int false "Página"
// @Param page_size query int false "Itens por página"
// @Success 200 {object} dto.UserListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users [get]
func (c *UserController) List(ctx *gin.Context) {
	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	pageStr := ctx.DefaultQuery("page", "0")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	// Ajustar a página para começar do 0 (frontend) para 1 (backend)
	page = page + 1

	pagination := dto.GetPagination(page, pageSize)

	// Calcular o offset para a paginação
	offset := (pagination.Page - 1) * pagination.PageSize

	// Buscar os usuários
	users, err := c.userRepository.List(ctx, tenantID, pagination.PageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao listar usuários", err.Error()))
		return
	}

	// Obter a contagem total
	totalCount, err := c.userRepository.CountByTenant(ctx, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao contar usuários", err.Error()))
		return
	}

	// Ajustar a página de volta para começar do 0 para o frontend
	pagination.Page = pagination.Page - 1

	// Montar a resposta
	response := dto.ToUserListResponse(users, totalCount, pagination.Page, pagination.PageSize)
	ctx.JSON(http.StatusOK, response)
}

// ListByBranch lista os usuários de uma filial
// @Summary Lista os usuários de uma filial
// @Description Lista os usuários de uma filial específica com paginação
// @Tags users
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param branch_id path string true "ID da filial"
// @Param page query int false "Página"
// @Param page_size query int false "Itens por página"
// @Success 200 {object} dto.UserListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/branch/{branch_id} [get]
func (c *UserController) ListByBranch(ctx *gin.Context) {
	branchID := ctx.Param("branch_id")
	if branchID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID da filial não fornecido", ""))
		return
	}

	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	pagination := dto.GetPagination(page, pageSize)

	// Calcular o offset para a paginação
	offset := (pagination.Page - 1) * pagination.PageSize

	// Buscar os usuários
	users, err := c.userRepository.FindByBranch(ctx, branchID, pagination.PageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao listar usuários da filial", err.Error()))
		return
	}

	// Obter a contagem total
	totalCount, err := c.userRepository.CountByBranch(ctx, branchID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao contar usuários da filial", err.Error()))
		return
	}

	// Montar a resposta
	response := dto.ToUserListResponse(users, totalCount, pagination.Page, pagination.PageSize)
	ctx.JSON(http.StatusOK, response)
}

// Update atualiza um usuário
// @Summary Atualiza um usuário
// @Description Atualiza os dados de um usuário existente
// @Tags users
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID do usuário"
// @Param user body dto.UserRequest true "Dados do usuário"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{id} [put]
func (c *UserController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Verificar se o usuário existe
	existingUser, err := c.userRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	// Fazer o bind dos dados da requisição
	var request dto.UserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Atualizar o usuário existente com os novos dados
	existingUser.Name = request.Name
	existingUser.Email = request.Email
	existingUser.BranchID = request.BranchID
	existingUser.Role = user.Role(request.Role)
	existingUser.UpdatedAt = time.Now()

	// Se uma senha foi fornecida, atualizar a senha
	if request.Password != "" {
		if err := existingUser.SetPassword(request.Password); err != nil {
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao processar senha", err.Error()))
			return
		}
	}

	// Persistir as alterações
	err = c.userRepository.Update(ctx, existingUser)
	if err != nil {
		if errors.Is(err, repository.ErrUserDuplicateEmail) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Email já está sendo usado por outro usuário", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar usuário", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToUserResponse(existingUser))
}

// ChangePassword altera a senha de um usuário
// @Summary Altera a senha de um usuário
// @Description Permite que um usuário altere sua própria senha
// @Tags users
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID do usuário"
// @Param password body dto.ChangePasswordRequest true "Dados para troca de senha"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{id}/password [patch]
func (c *UserController) ChangePassword(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Verificar se o usuário existe
	existingUser, err := c.userRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	// Verificar se o usuário atual é o mesmo que está sendo alterado ou é um administrador
	userID, _, _, _, userRole, _ := auth.GetCurrentUser(ctx)
	if userID != id && userRole != string(user.RoleAdmin) {
		ctx.JSON(http.StatusForbidden, dto.NewErrorResponse(http.StatusForbidden, "Permissão negada", "Você só pode alterar sua própria senha"))
		return
	}

	// Fazer o bind dos dados da requisição
	var request dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Verificar se a senha atual está correta (exceto para administradores alterando outro usuário)
	if userID == id && !existingUser.CheckPassword(request.CurrentPassword) {
		ctx.JSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "Senha atual incorreta", ""))
		return
	}

	// Gerar hash da nova senha
	if err := existingUser.SetPassword(request.NewPassword); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao processar senha", err.Error()))
		return
	}

	// Atualizar a senha no banco de dados
	err = c.userRepository.UpdatePassword(ctx, id, existingUser.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar senha", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("Senha atualizada com sucesso", nil))
}

// Delete remove um usuário
// @Summary Remove um usuário
// @Description Remove um usuário do sistema
// @Tags users
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID do usuário"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{id} [delete]
func (c *UserController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Verificar se o usuário existe
	_, err := c.userRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário", err.Error()))
		return
	}

	// Verificar se o usuário não está tentando excluir a si mesmo
	userID, _, _, _, _, _ := auth.GetCurrentUser(ctx)
	if userID == id {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Operação não permitida", "Você não pode excluir seu próprio usuário"))
		return
	}

	// Remover o usuário
	err = c.userRepository.Delete(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao remover usuário", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("Usuário removido com sucesso", nil))
}

// UpdateStatus atualiza o status de um usuário
// @Summary Atualiza o status de um usuário
// @Description Atualiza o status de um usuário (ativo/inativo/bloqueado)
// @Tags users
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID do usuário"
// @Param status path string true "Novo status (active/inactive/blocked)"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /users/{id}/status/{status} [patch]
func (c *UserController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	statusStr := ctx.Param("status")

	if id == "" || statusStr == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID ou status não fornecido", ""))
		return
	}

	var status user.Status
	switch statusStr {
	case "active":
		status = user.StatusActive
	case "inactive":
		status = user.StatusInactive
	case "blocked":
		status = user.StatusBlocked
	default:
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Status inválido", "Valores aceitos: active, inactive, blocked"))
		return
	}

	// Verificar se o usuário não está tentando alterar o status de si mesmo
	userID, _, _, _, _, _ := auth.GetCurrentUser(ctx)
	if userID == id {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Operação não permitida", "Você não pode alterar seu próprio status"))
		return
	}

	// Atualizar status
	err := c.userRepository.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Usuário não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar status", err.Error()))
		return
	}

	// Buscar o usuário atualizado
	u, err := c.userRepository.FindByID(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar usuário atualizado", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToUserResponse(u))
}

// CreateAdminUser cria o primeiro usuário administrador para um tenant
// @Summary Cria o primeiro usuário administrador
// @Description Cria o primeiro usuário administrador para um tenant (não requer autenticação)
// @Tags setup
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param user body dto.UserRequest true "Dados do usuário administrador"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /setup/admin [post]
func (c *UserController) CreateAdminUser(ctx *gin.Context) {
	var request dto.UserRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Verificar se a senha foi fornecida (obrigatória para novos usuários)
	if request.Password == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Senha requerida", "A senha é obrigatória para novos usuários"))
		return
	}

	// Verificar se o Role foi fornecido e definir como ADMIN se não foi
	if request.Role == "" {
		request.Role = string(user.RoleAdmin)
	}

	// Obter tenant ID do cabeçalho
	tenantID := ctx.GetHeader("tenant-id")
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não fornecido", "O cabeçalho 'tenant-id' é obrigatório"))
		return
	}

	// Verificar se o tenant existe
	tenantExists, err := c.userRepository.TenantExists(ctx, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao verificar tenant", err.Error()))
		return
	}
	if !tenantExists {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant inválido", "O tenant informado não existe"))
		return
	}

	// Verificar se já existe algum usuário para este tenant
	usersCount, err := c.userRepository.CountByTenant(ctx, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao verificar usuários existentes", err.Error()))
		return
	}
	if usersCount > 0 {
		ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Usuário administrador já existe", "Já existe pelo menos um usuário para este tenant"))
		return
	}

	// Forçar o papel como admin
	request.Role = string(user.RoleAdmin)

	// Garantir que branch_id não seja uma string vazia
	branchID := request.BranchID
	if branchID == "" {
		branchID = "" // Mantem vazio e será tratado como NULL no repositório
	}

	// Gerar um ID para o novo usuário
	id := uuid.New().String()

	// Criar o modelo de domínio a partir do DTO
	u := &user.User{
		ID:          id,
		TenantID:    tenantID,
		BranchID:    branchID,
		Name:        request.Name,
		Email:       request.Email,
		Role:        user.RoleAdmin, // Forçar como admin
		Status:      user.StatusActive,
		LastLoginAt: time.Time{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Definir a senha com hash
	if err := u.SetPassword(request.Password); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao processar senha", err.Error()))
		return
	}

	// Persistir o usuário
	err = c.userRepository.Create(ctx, u)
	if err != nil {
		if errors.Is(err, repository.ErrUserDuplicateEmail) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Usuário com mesmo email já existe", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao criar usuário", err.Error()))
		return
	}

	// Retornar o usuário criado
	ctx.JSON(http.StatusCreated, dto.ToUserResponse(u))
}
