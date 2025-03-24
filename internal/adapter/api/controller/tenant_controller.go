package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	"github.com/hugohenrick/erp-supermercado/internal/domain/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantController gerencia as requisições relacionadas aos tenants
type TenantController struct {
	tenantRepository tenant.Repository
	db               *pgxpool.Pool
}

// NewTenantController cria uma nova instância de TenantController
func NewTenantController(tenantRepository tenant.Repository, db *pgxpool.Pool) *TenantController {
	return &TenantController{
		tenantRepository: tenantRepository,
		db:               db,
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
		if err == repository.ErrTenantDuplicateDocument {
			ctx.JSON(http.StatusConflict, dto.NewErrorResponse(http.StatusConflict, "Tenant já existe", "Um tenant com este documento já está cadastrado"))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "Erro ao criar tenant", err.Error()))
		return
	}

	// Criar o schema para o tenant no banco de dados
	err = c.createTenantSchema(ctx, id, schema)
	if err != nil {
		// Se falhar ao criar o schema, excluir o tenant para manter a consistência
		deleteErr := c.tenantRepository.Delete(ctx, id)
		if deleteErr != nil {
			// Logar o erro de exclusão, mas continuar com o erro principal
			// Em um ambiente de produção, isso deveria ser registrado em um sistema de logs
			// logger.Error("Falha ao excluir tenant após falha na criação do schema", "error", deleteErr)
		}

		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(
			http.StatusInternalServerError,
			"Erro ao criar schema do tenant",
			err.Error(),
		))
		return
	}

	// Retornar o tenant criado
	response := dto.ToTenantResponse(t)
	ctx.JSON(http.StatusCreated, response)
}

// createTenantSchema cria um novo schema no banco de dados para o tenant
func (c *TenantController) createTenantSchema(ctx context.Context, tenantID, schema string) error {
	conn, err := c.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("erro ao adquirir conexão do pool: %w", err)
	}
	defer conn.Release()

	// Criar schema
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schema))
	if err != nil {
		return fmt.Errorf("erro ao criar schema: %w", err)
	}

	// Configurar permissões
	_, err = conn.Exec(ctx, fmt.Sprintf("GRANT ALL ON SCHEMA %s TO CURRENT_USER", schema))
	if err != nil {
		return fmt.Errorf("erro ao configurar permissões do schema: %w", err)
	}

	// Aplicar migrações no schema do tenant
	err = c.applyTenantMigrations(ctx, schema)
	if err != nil {
		return fmt.Errorf("erro ao aplicar migrações no schema do tenant: %w", err)
	}

	return nil
}

// applyTenantMigrations executa as migrações necessárias no schema do tenant
func (c *TenantController) applyTenantMigrations(ctx context.Context, schema string) error {
	// Lista de migrações a serem aplicadas, na ordem correta
	migrations := []struct {
		description string
		filename    string
	}{
		{
			description: "Criação da tabela de filiais (branches)",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000001_create_branches_table.up.sql",
		},
		{
			description: "Criação da tabela de usuários (users)",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000002_create_users_table.up.sql",
		},
		{
			description: "Criação da tabela de categorias de produtos",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000003_create_product_categories_table.up.sql",
		},
		{
			description: "Criação da tabela de produtos",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000004_create_products_table.up.sql",
		},
		{
			description: "Criação das tabelas de estoque",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000005_create_inventory_tables.up.sql",
		},
		{
			description: "Criação da tabela de clientes",
			filename:    "/home/hugohenrick/Estudos/Cursor/super/erp-supermercado/migrations/tenant/000006_create_customers_table.up.sql",
		},
	}

	conn, err := c.db.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("erro ao adquirir conexão do pool: %w", err)
	}
	defer conn.Release()

	// Tabela para rastrear migrações aplicadas neste schema
	_, err = conn.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s.schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`, schema))
	if err != nil {
		return fmt.Errorf("erro ao criar tabela de controle de migrações: %w", err)
	}

	// Para cada migração, verificar se já foi aplicada e aplicar se necessário
	for _, migration := range migrations {
		// Verificar se a migração já foi aplicada
		var exists bool
		err = conn.QueryRow(ctx, fmt.Sprintf(`
			SELECT EXISTS(
				SELECT 1 FROM %s.schema_migrations WHERE version = $1
			)`, schema), path.Base(migration.filename)).Scan(&exists)

		if err != nil {
			return fmt.Errorf("erro ao verificar migração: %w", err)
		}

		// Se a migração já foi aplicada, pular
		if exists {
			continue
		}

		// Ler o conteúdo do arquivo de migração
		sqlContent, err := os.ReadFile(migration.filename)
		if err != nil {
			return fmt.Errorf("erro ao ler arquivo de migração %s: %w", migration.filename, err)
		}

		// Substituir referências ao schema padrão "public" pelo schema do tenant
		sqlString := string(sqlContent)

		// Iniciar uma transação para aplicar a migração
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("erro ao iniciar transação: %w", err)
		}

		// Aplicar a migração, configurando o schema de busca
		_, err = tx.Exec(ctx, fmt.Sprintf("SET search_path TO %s", schema))
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("erro ao configurar search_path: %w", err)
		}

		// Executar o script SQL da migração
		_, err = tx.Exec(ctx, sqlString)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("erro ao aplicar migração %s: %w", migration.filename, err)
		}

		// Registrar a migração como aplicada
		_, err = tx.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s.schema_migrations (version) VALUES ($1)
		`, schema), path.Base(migration.filename))

		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("erro ao registrar migração: %w", err)
		}

		// Commit da transação
		err = tx.Commit(ctx)
		if err != nil {
			return fmt.Errorf("erro ao fazer commit da transação: %w", err)
		}
	}

	return nil
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
