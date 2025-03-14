package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/repository"
	customerdomain "github.com/hugohenrick/erp-supermercado/internal/domain/customer"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// CustomerController gerencia as requisições relacionadas a clientes
type CustomerController struct {
	customerRepo customerdomain.Repository
	logger       logger.Logger
}

// NewCustomerController cria uma nova instância de CustomerController
func NewCustomerController(customerRepo customerdomain.Repository, logger logger.Logger) *CustomerController {
	return &CustomerController{
		customerRepo: customerRepo,
		logger:       logger,
	}
}

// Create cria um novo cliente
// @Summary Criar cliente
// @Description Cria um novo cliente no sistema
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param customer body dto.CustomerRequest true "Dados do cliente"
// @Success 201 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers [post]
func (c *CustomerController) Create(ctx *gin.Context) {
	var req dto.CustomerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	tenantID := ctx.GetString("tenant_id")
	branchID := ctx.GetString("branch_id")

	customer, err := customerdomain.NewCustomer(
		tenantID,
		branchID,
		req.PersonType,
		req.Name,
		req.Document,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao criar cliente", err.Error()))
		return
	}

	// Atualizar dados adicionais
	err = customer.Update(
		req.Name,
		req.TradeName,
		req.StateDocument,
		req.CityDocument,
		req.TaxRegime,
		req.CustomerType,
		req.CreditLimit,
		req.PaymentTerm,
		req.Website,
		req.Observations,
		req.FiscalNotes,
		req.ExternalCode,
		req.SalesmanID,
		req.PriceTableID,
		req.PaymentMethodID,
		req.SUFRAMA,
		req.ReferenceCode,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao atualizar dados do cliente", err.Error()))
		return
	}

	// Adicionar endereços
	for _, addr := range req.Addresses {
		address := customerdomain.Address{
			Street:          addr.Street,
			Number:          addr.Number,
			Complement:      addr.Complement,
			District:        addr.District,
			City:            addr.City,
			State:           addr.State,
			ZipCode:         addr.ZipCode,
			Country:         addr.Country,
			AddressType:     addr.AddressType,
			MainAddress:     false,
			DeliveryAddress: false,
		}
		customer.AddAddress(address)
	}

	// Adicionar contatos
	for _, cont := range req.Contacts {
		contact := customerdomain.Contact{
			Name:        cont.Name,
			Department:  cont.Department,
			Phone:       cont.Phone,
			MobilePhone: cont.MobilePhone,
			Email:       cont.Email,
			Position:    cont.Position,
			MainContact: cont.MainContact,
		}
		customer.AddContact(contact)
	}

	if err := c.customerRepo.Create(ctx, customer); err != nil {
		c.logger.Error("erro ao criar cliente no banco de dados", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao salvar cliente", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, dto.ToCustomerResponse(customer))
}

// Get retorna um cliente pelo ID
// @Summary Buscar cliente
// @Description Retorna os dados de um cliente pelo ID
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do cliente"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/{id} [get]
func (c *CustomerController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "id não informado", ""))
		return
	}

	customer, err := c.customerRepo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrCustomerNotFound {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "cliente não encontrado", err.Error()))
			return
		}
		c.logger.Error("erro ao buscar cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao buscar cliente", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToCustomerResponse(customer))
}

// List retorna a lista de clientes
// @Summary Listar clientes
// @Description Retorna a lista de clientes paginada
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Número da página"
// @Param size query int false "Tamanho da página"
// @Success 200 {object} dto.CustomerListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers [get]
func (c *CustomerController) List(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	offset := (page - 1) * size

	tenantID := tenant.GetTenantID(ctx)

	customers, err := c.customerRepo.List(ctx, tenantID, size, offset)
	if err != nil {
		c.logger.Error("erro ao listar clientes", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar clientes", err.Error()))
		return
	}

	total, err := c.customerRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		c.logger.Error("erro ao contar clientes", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao contar clientes", err.Error()))
		return
	}

	totalPages := (total + size - 1) / size

	ctx.JSON(http.StatusOK, dto.ToCustomerListResponse(customers, total, page, size, totalPages))
}

// Update atualiza um cliente
// @Summary Atualizar cliente
// @Description Atualiza os dados de um cliente
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do cliente"
// @Param customer body dto.CustomerRequest true "Dados do cliente"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/{id} [put]
func (c *CustomerController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "id não informado", ""))
		return
	}

	var req dto.CustomerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	customer, err := c.customerRepo.FindByID(ctx, id)
	if err != nil {
		if err == repository.ErrCustomerNotFound {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "cliente não encontrado", err.Error()))
			return
		}
		c.logger.Error("erro ao buscar cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao buscar cliente", err.Error()))
		return
	}

	// Atualizar dados
	err = customer.Update(
		req.Name,
		req.TradeName,
		req.StateDocument,
		req.CityDocument,
		req.TaxRegime,
		req.CustomerType,
		req.CreditLimit,
		req.PaymentTerm,
		req.Website,
		req.Observations,
		req.FiscalNotes,
		req.ExternalCode,
		req.SalesmanID,
		req.PriceTableID,
		req.PaymentMethodID,
		req.SUFRAMA,
		req.ReferenceCode,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao atualizar dados do cliente", err.Error()))
		return
	}

	// Atualizar endereços
	customer.Addresses = nil
	for _, addr := range req.Addresses {
		address := customerdomain.Address{
			Street:          addr.Street,
			Number:          addr.Number,
			Complement:      addr.Complement,
			District:        addr.District,
			City:            addr.City,
			State:           addr.State,
			ZipCode:         addr.ZipCode,
			Country:         addr.Country,
			AddressType:     addr.AddressType,
			MainAddress:     false,
			DeliveryAddress: false,
		}
		customer.AddAddress(address)
	}

	// Atualizar contatos
	customer.Contacts = nil
	for _, cont := range req.Contacts {
		contact := customerdomain.Contact{
			Name:        cont.Name,
			Department:  cont.Department,
			Phone:       cont.Phone,
			MobilePhone: cont.MobilePhone,
			Email:       cont.Email,
			Position:    cont.Position,
			MainContact: cont.MainContact,
		}
		customer.AddContact(contact)
	}

	if err := c.customerRepo.Update(ctx, customer); err != nil {
		c.logger.Error("erro ao atualizar cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar cliente", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToCustomerResponse(customer))
}

// Delete exclui um cliente
// @Summary Excluir cliente
// @Description Exclui um cliente do sistema
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do cliente"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/{id} [delete]
func (c *CustomerController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "id não informado", ""))
		return
	}

	if err := c.customerRepo.Delete(ctx, id); err != nil {
		if err == repository.ErrCustomerNotFound {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "cliente não encontrado", err.Error()))
			return
		}
		c.logger.Error("erro ao excluir cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao excluir cliente", err.Error()))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// UpdateStatus atualiza o status de um cliente
// @Summary Atualizar status do cliente
// @Description Atualiza o status de um cliente
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do cliente"
// @Param status body string true "Novo status"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/{id}/status [patch]
func (c *CustomerController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "id não informado", ""))
		return
	}

	var status customerdomain.Status
	if err := ctx.ShouldBindJSON(&status); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "status inválido", err.Error()))
		return
	}

	if err := c.customerRepo.UpdateStatus(ctx, id, status); err != nil {
		if err == repository.ErrCustomerNotFound {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "cliente não encontrado", err.Error()))
			return
		}
		c.logger.Error("erro ao atualizar status do cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar status do cliente", err.Error()))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// FindByDocument busca um cliente pelo documento
// @Summary Buscar cliente por documento
// @Description Retorna os dados de um cliente pelo documento (CPF/CNPJ)
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param document path string true "Documento do cliente"
// @Success 200 {object} dto.CustomerResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/document/{document} [get]
func (c *CustomerController) FindByDocument(ctx *gin.Context) {
	document := ctx.Param("document")
	if document == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "documento não informado", ""))
		return
	}

	tenantID := ctx.GetString("tenant_id")

	customer, err := c.customerRepo.FindByDocument(ctx, tenantID, document)
	if err != nil {
		if err == repository.ErrCustomerNotFound {
			ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "cliente não encontrado", err.Error()))
			return
		}
		c.logger.Error("erro ao buscar cliente", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao buscar cliente", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.ToCustomerResponse(customer))
}

// FindByName busca clientes pelo nome
// @Summary Buscar clientes por nome
// @Description Retorna a lista de clientes que contêm o nome informado
// @Tags customers
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param name query string true "Nome do cliente"
// @Param page query int false "Número da página"
// @Param size query int false "Tamanho da página"
// @Success 200 {object} dto.CustomerListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /customers/search [get]
func (c *CustomerController) FindByName(ctx *gin.Context) {
	name := ctx.Query("name")
	if name == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "nome não informado", ""))
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "10"))

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}

	offset := (page - 1) * size

	tenantID := ctx.GetString("tenant_id")

	customers, err := c.customerRepo.FindByName(ctx, tenantID, name, size, offset)
	if err != nil {
		c.logger.Error("erro ao buscar clientes", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao buscar clientes", err.Error()))
		return
	}

	total, err := c.customerRepo.CountByTenant(ctx, tenantID)
	if err != nil {
		c.logger.Error("erro ao contar clientes", "error", err)
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao contar clientes", err.Error()))
		return
	}

	totalPages := (total + size - 1) / size

	ctx.JSON(http.StatusOK, dto.ToCustomerListResponse(customers, total, page, size, totalPages))
}
