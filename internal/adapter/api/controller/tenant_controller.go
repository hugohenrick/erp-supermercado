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
	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
)

// TenantController gerencia as requisições relacionadas a tenants
type TenantController struct {
	tenantRepository tenant.Repository
}

// NewTenantController cria uma nova instância de TenantController
func NewTenantController(tenantRepository tenant.Repository) *TenantController {
	return &TenantController{
		tenantRepository: tenantRepository,
	}
}

// Create cria um novo tenant
// @Summary Cria um novo tenant
// @Description Cria um novo tenant no sistema
// @Tags tenants
// @Accept json
// @Produce json
// @Param tenant body dto.TenantRequest true "Dados do tenant"
// @Success 201 {object} dto.TenantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants [post]
func (c *TenantController) Create(ctx *gin.Context) {
	var request dto.TenantRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Gerar um ID para o novo tenant
	id := uuid.New().String()

	// Gerar um nome de schema único baseado no ID
	schema := "tenant_" + id[:8]

	// Criar o modelo de domínio a partir do DTO
	t := &tenant.Tenant{
		ID:          id,
		Name:        request.Name,
		Document:    request.Document,
		Email:       request.Email,
		Phone:       request.Phone,
		Status:      tenant.StatusActive, // Por padrão, o tenant é criado ativo
		Schema:      schema,
		PlanType:    request.PlanType,
		MaxBranches: request.MaxBranches,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Persistir o tenant
	err := c.tenantRepository.Create(ctx, t)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Tenant com mesmo documento já existe", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao criar tenant", err.Error()))
		return
	}

	// Retornar o tenant criado
	ctx.JSON(http.StatusCreated, dto.ToTenantResponse(t))
}

// GetByID busca um tenant pelo ID
// @Summary Busca um tenant pelo ID
// @Description Busca um tenant pelo seu ID
// @Tags tenants
// @Produce json
// @Param id path string true "ID do tenant"
// @Success 200 {object} dto.TenantResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants/{id} [get]
func (c *TenantController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	t, err := c.tenantRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Tenant não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar tenant", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToTenantResponse(t))
}

// GetByDocument busca um tenant pelo documento
// @Summary Busca um tenant pelo documento
// @Description Busca um tenant pelo seu documento (CNPJ/CPF)
// @Tags tenants
// @Produce json
// @Param document path string true "Documento do tenant"
// @Success 200 {object} dto.TenantResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants/document/{document} [get]
func (c *TenantController) GetByDocument(ctx *gin.Context) {
	document := ctx.Param("document")
	if document == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Documento não fornecido", ""))
		return
	}

	t, err := c.tenantRepository.FindByDocument(ctx, document)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Tenant não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar tenant", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToTenantResponse(t))
}

// List lista os tenants com paginação
// @Summary Lista os tenants
// @Description Lista os tenants com paginação
// @Tags tenants
// @Produce json
// @Param page query int false "Página"
// @Param page_size query int false "Itens por página"
// @Success 200 {object} dto.TenantListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants [get]
func (c *TenantController) List(ctx *gin.Context) {
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	pagination := dto.GetPagination(page, pageSize)

	// Calcular o offset para a paginação
	offset := (pagination.Page - 1) * pagination.PageSize

	// Buscar os tenants
	tenants, err := c.tenantRepository.List(ctx, pagination.PageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao listar tenants", err.Error()))
		return
	}

	// Obter a contagem total
	totalCount, err := c.tenantRepository.Count(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao contar tenants", err.Error()))
		return
	}

	// Montar a resposta
	response := dto.ToTenantListResponse(tenants, totalCount, pagination.Page, pagination.PageSize)
	ctx.JSON(http.StatusOK, response)
}

// Update atualiza um tenant
// @Summary Atualiza um tenant
// @Description Atualiza os dados de um tenant existente
// @Tags tenants
// @Accept json
// @Produce json
// @Param id path string true "ID do tenant"
// @Param tenant body dto.TenantRequest true "Dados do tenant"
// @Success 200 {object} dto.TenantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants/{id} [put]
func (c *TenantController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Verificar se o tenant existe
	existingTenant, err := c.tenantRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Tenant não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar tenant", err.Error()))
		return
	}

	// Fazer o bind dos dados da requisição
	var request dto.TenantRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Atualizar o tenant existente com os novos dados
	existingTenant.Name = request.Name
	existingTenant.Email = request.Email
	existingTenant.Phone = request.Phone
	existingTenant.PlanType = request.PlanType
	existingTenant.MaxBranches = request.MaxBranches
	existingTenant.UpdatedAt = time.Now()

	// Não permitimos alterar o documento (CNPJ/CPF)

	// Persistir as alterações
	err = c.tenantRepository.Update(ctx, existingTenant)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar tenant", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToTenantResponse(existingTenant))
}

// Delete remove um tenant
// @Summary Remove um tenant
// @Description Remove um tenant do sistema
// @Tags tenants
// @Produce json
// @Param id path string true "ID do tenant"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants/{id} [delete]
func (c *TenantController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Verificar se o tenant existe
	_, err := c.tenantRepository.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Tenant não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar tenant", err.Error()))
		return
	}

	// Remover o tenant
	err = c.tenantRepository.Delete(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao remover tenant", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("Tenant removido com sucesso", nil))
}

// UpdateStatus atualiza o status de um tenant
// @Summary Atualiza o status de um tenant
// @Description Atualiza o status de um tenant (ativo/inativo)
// @Tags tenants
// @Produce json
// @Param id path string true "ID do tenant"
// @Param status path string true "Novo status (active/inactive)"
// @Success 200 {object} dto.TenantResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /tenants/{id}/status/{status} [patch]
func (c *TenantController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	statusStr := ctx.Param("status")

	if id == "" || statusStr == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID ou status não fornecido", ""))
		return
	}

	var status tenant.Status
	switch statusStr {
	case "active":
		status = tenant.StatusActive
	case "inactive":
		status = tenant.StatusInactive
	default:
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Status inválido", "Valores aceitos: active, inactive"))
		return
	}

	// Atualizar status
	err := c.tenantRepository.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, repository.ErrTenantNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Tenant não encontrado", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar status", err.Error()))
		return
	}

	// Buscar o tenant atualizado
	t, err := c.tenantRepository.FindByID(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar tenant atualizado", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToTenantResponse(t))
}
