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
	"github.com/hugohenrick/erp-supermercado/internal/domain/branch"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// BranchController gerencia as requisições relacionadas a filiais
type BranchController struct {
	branchRepository branch.Repository
}

// NewBranchController cria uma nova instância de BranchController
func NewBranchController(branchRepository branch.Repository) *BranchController {
	return &BranchController{
		branchRepository: branchRepository,
	}
}

// Create cria uma nova filial
// @Summary Cria uma nova filial
// @Description Cria uma nova filial no sistema
// @Tags branches
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param branch body dto.BranchRequest true "Dados da filial"
// @Success 201 {object} dto.BranchResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 409 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches [post]
func (c *BranchController) Create(ctx *gin.Context) {
	var request dto.BranchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	// Gerar um ID para a nova filial
	id := uuid.New().String()

	// Criar o modelo de domínio a partir do DTO
	b := &branch.Branch{
		ID:       id,
		TenantID: tenantID,
		Name:     request.Name,
		Code:     request.Code,
		Type:     branch.BranchType(request.Type),
		Document: request.Document,
		Phone:    request.Phone,
		Email:    request.Email,
		Address: branch.Address{
			Street:     request.Address.Street,
			Number:     request.Address.Number,
			Complement: request.Address.Complement,
			District:   request.Address.District,
			City:       request.Address.City,
			State:      request.Address.State,
			ZipCode:    request.Address.ZipCode,
			Country:    request.Address.Country,
		},
		Status:    branch.StatusActive, // Por padrão, a filial é criada ativa
		IsMain:    request.IsMain,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Persistir a filial
	err := c.branchRepository.Create(ctx, b)
	if err != nil {
		if errors.Is(err, repository.ErrBranchDuplicateKey) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Filial com mesmo código já existe para este tenant", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao criar filial", err.Error()))
		return
	}

	// Retornar a filial criada
	ctx.JSON(http.StatusCreated, dto.ToBranchResponse(b))
}

// GetByID busca uma filial pelo ID
// @Summary Busca uma filial pelo ID
// @Description Busca uma filial pelo seu ID
// @Tags branches
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID da filial"
// @Success 200 {object} dto.BranchResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches/{id} [get]
func (c *BranchController) GetByID(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	b, err := c.branchRepository.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, repository.ErrBranchNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Filial não encontrada", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar filial", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToBranchResponse(b))
}

// GetMainBranch busca a filial principal do tenant
// @Summary Busca a filial principal
// @Description Busca a filial principal do tenant
// @Tags branches
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Success 200 {object} dto.BranchResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches/main [get]
func (c *BranchController) GetMainBranch(ctx *gin.Context) {
	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	b, err := c.branchRepository.FindMainBranch(ctx, tenantID)
	if err != nil {
		if errors.Is(err, repository.ErrBranchNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Filial principal não encontrada", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar filial principal", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToBranchResponse(b))
}

// List lista as filiais com paginação
// @Summary Lista as filiais
// @Description Lista as filiais do tenant com paginação
// @Tags branches
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param page query int false "Página"
// @Param page_size query int false "Itens por página"
// @Success 200 {object} dto.BranchListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches [get]
func (c *BranchController) List(ctx *gin.Context) {
	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("page_size", "10")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	pagination := dto.GetPagination(page, pageSize)

	// Calcular o offset para a paginação
	offset := (pagination.Page - 1) * pagination.PageSize

	// Buscar as filiais
	branches, err := c.branchRepository.ListByTenant(ctx, tenantID, pagination.PageSize, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao listar filiais", err.Error()))
		return
	}

	// Obter a contagem total
	totalCount, err := c.branchRepository.CountByTenant(ctx, tenantID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao contar filiais", err.Error()))
		return
	}

	// Montar a resposta
	response := dto.ToBranchListResponse(branches, totalCount, pagination.Page, pagination.PageSize)
	ctx.JSON(http.StatusOK, response)
}

// Update atualiza uma filial
// @Summary Atualiza uma filial
// @Description Atualiza os dados de uma filial existente
// @Tags branches
// @Accept json
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID da filial"
// @Param branch body dto.BranchRequest true "Dados da filial"
// @Success 200 {object} dto.BranchResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches/{id} [put]
func (c *BranchController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	if tenantID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Tenant ID não encontrado", ""))
		return
	}

	// Verificar se a filial existe
	existingBranch, err := c.branchRepository.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, repository.ErrBranchNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Filial não encontrada", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar filial", err.Error()))
		return
	}

	// Fazer o bind dos dados da requisição
	var request dto.BranchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Requisição inválida", err.Error()))
		return
	}

	// Atualizar a filial existente com os novos dados
	existingBranch.Name = request.Name
	existingBranch.Code = request.Code
	existingBranch.Type = branch.BranchType(request.Type)
	existingBranch.Document = request.Document
	existingBranch.Phone = request.Phone
	existingBranch.Email = request.Email
	existingBranch.Address = branch.Address{
		Street:     request.Address.Street,
		Number:     request.Address.Number,
		Complement: request.Address.Complement,
		District:   request.Address.District,
		City:       request.Address.City,
		State:      request.Address.State,
		ZipCode:    request.Address.ZipCode,
		Country:    request.Address.Country,
	}
	existingBranch.IsMain = request.IsMain
	existingBranch.UpdatedAt = time.Now()

	// Persistir as alterações
	err = c.branchRepository.Update(ctx, existingBranch)
	if err != nil {
		if errors.Is(err, repository.ErrBranchDuplicateKey) {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Filial com mesmo código já existe para este tenant", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar filial", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToBranchResponse(existingBranch))
}

// Delete remove uma filial
// @Summary Remove uma filial
// @Description Remove uma filial do sistema
// @Tags branches
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID da filial"
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches/{id} [delete]
func (c *BranchController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID não fornecido", ""))
		return
	}

	// Remover a filial
	err := c.branchRepository.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrBranchNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Filial não encontrada", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao remover filial", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("Filial removida com sucesso", nil))
}

// UpdateStatus atualiza o status de uma filial
// @Summary Atualiza o status de uma filial
// @Description Atualiza o status de uma filial (ativa/inativa)
// @Tags branches
// @Produce json
// @Param tenant-id header string true "ID do tenant"
// @Param id path string true "ID da filial"
// @Param status path string true "Novo status (active/inactive)"
// @Success 200 {object} dto.BranchResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /branches/{id}/status/{status} [patch]
func (c *BranchController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	statusStr := ctx.Param("status")

	if id == "" || statusStr == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID ou status não fornecido", ""))
		return
	}

	var status branch.Status
	switch statusStr {
	case "active":
		status = branch.StatusActive
	case "inactive":
		status = branch.StatusInactive
	default:
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "Status inválido", "Valores aceitos: active, inactive"))
		return
	}

	// Atualizar status
	err := c.branchRepository.UpdateStatus(ctx, id, status)
	if err != nil {
		if errors.Is(err, repository.ErrBranchNotFound) {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "Filial não encontrada", ""))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao atualizar status", err.Error()))
		return
	}

	// Buscar a filial atualizada
	// Obter tenant ID do contexto
	tenantID := tenant.GetTenantID(ctx)
	b, err := c.branchRepository.FindByTenantAndID(ctx, tenantID, id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao buscar filial atualizada", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToBranchResponse(b))
}
