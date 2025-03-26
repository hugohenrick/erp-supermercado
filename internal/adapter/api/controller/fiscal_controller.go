package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/domain/fiscal"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
)

// FiscalController manipula as requisições relacionadas às configurações fiscais
type FiscalController struct {
	fiscalRepo fiscal.Repository
	logger     logger.Logger
}

// NewFiscalController cria uma nova instância de FiscalController
func NewFiscalController(fiscalRepo fiscal.Repository, logger logger.Logger) *FiscalController {
	return &FiscalController{
		fiscalRepo: fiscalRepo,
		logger:     logger,
	}
}

// @Summary Criar configuração fiscal
// @Description Cria uma nova configuração fiscal
// @Tags Configurações Fiscais
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param config body dto.FiscalConfigRequest true "Dados da configuração fiscal"
// @Success 201 {object} dto.FiscalConfigResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs [post]
func (c *FiscalController) Create(ctx *gin.Context) {
	var req dto.FiscalConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	tenantID := ctx.GetString("tenant_id")

	// Criar a configuração fiscal
	config, err := fiscal.NewConfiguration(tenantID, req.BranchID, req.CertificateID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao criar configuração fiscal", err.Error()))
		return
	}

	// Configurar dados de NFe
	err = config.ConfigureNFe(
		req.NFeSeries,
		req.NFeNextNumber,
		req.NFeEnvironment,
		req.NFeCSCID,
		req.NFeCSCToken,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar NFe", err.Error()))
		return
	}

	// Configurar dados de NFCe
	err = config.ConfigureNFCe(
		req.NFCeSeries,
		req.NFCeNextNumber,
		req.NFCeEnvironment,
		req.NFCeCSCID,
		req.NFCeCSCToken,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar NFCe", err.Error()))
		return
	}

	// Configurar SMTP
	err = config.ConfigureSMTP(
		req.SMTPHost,
		req.SMTPPort,
		req.SMTPUsername,
		req.SMTPPassword,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar SMTP", err.Error()))
		return
	}

	// Configurar impressão
	config.ConfigurePrinting(
		req.PrintDANFEMode,
		req.PrinterName,
		req.PrinterPaperSize,
	)

	// Configurar contingência
	if req.ContingencyEnabled {
		config.EnableContingency()
	}

	// Salvar a configuração
	if err := c.fiscalRepo.Create(ctx, config); err != nil {
		c.logger.Error("erro ao salvar configuração fiscal", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao salvar configuração fiscal", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewFiscalConfigResponse(config))
}

// @Summary Obter configuração fiscal
// @Description Busca uma configuração fiscal pelo ID
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID da configuração fiscal"
// @Success 200 {object} dto.FiscalConfigResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/{id} [get]
func (c *FiscalController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID da configuração fiscal não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Buscar a configuração fiscal no repositório
	config, err := c.fiscalRepo.FindByID(ctx, id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "erro ao buscar configuração fiscal"
		if err.Error() == fmt.Sprintf("configuração fiscal com ID %s não encontrada", id) {
			statusCode = http.StatusNotFound
			errorMsg = "configuração fiscal não encontrada"
		}
		ctx.JSON(statusCode, dto.NewErrorResponse(statusCode, errorMsg, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewFiscalConfigResponse(config))
}

// @Summary Listar configurações fiscais
// @Description Lista as configurações fiscais com paginação
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Número da página (padrão: 1)"
// @Param page_size query int false "Tamanho da página (padrão: 10)"
// @Param branch_id query string false "Filtrar por filial"
// @Success 200 {object} dto.FiscalConfigListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs [get]
func (c *FiscalController) List(ctx *gin.Context) {
	tenantID := ctx.GetString("tenant_id")
	branchID := ctx.Query("branch_id")

	// Parâmetros de paginação
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	var configs []*fiscal.Configuration
	var err error
	var total int

	// Se branch_id foi fornecido, buscar apenas a configuração dessa filial
	if branchID != "" {
		config, err := c.fiscalRepo.FindByBranch(ctx, branchID)
		if err != nil {
			c.logger.Error("erro ao buscar configuração fiscal", "error", err.Error())
			if err.Error() == fmt.Sprintf("configuração fiscal para filial %s não encontrada", branchID) {
				configs = []*fiscal.Configuration{}
				total = 0
			} else {
				ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao buscar configuração fiscal", err.Error()))
				return
			}
		} else {
			configs = []*fiscal.Configuration{config}
			total = 1
		}
	} else {
		// Calcular offset para paginação
		offset := (page - 1) * pageSize

		// Listar todas as configurações do tenant
		configs, err = c.fiscalRepo.List(ctx, tenantID, pageSize, offset)
		if err != nil {
			c.logger.Error("erro ao listar configurações fiscais", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar configurações fiscais", err.Error()))
			return
		}

		// Contar total de configurações
		total = len(configs)
	}

	ctx.JSON(http.StatusOK, dto.NewFiscalConfigListResponse(configs, total, page, pageSize))
}

// @Summary Atualizar configuração fiscal
// @Description Atualiza uma configuração fiscal existente
// @Tags Configurações Fiscais
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID da configuração fiscal"
// @Param config body dto.FiscalConfigRequest true "Dados da configuração fiscal"
// @Success 200 {object} dto.FiscalConfigResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/{id} [put]
func (c *FiscalController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	var req dto.FiscalConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	// Buscar configuração existente
	existingConfig, err := c.fiscalRepo.FindByID(ctx, id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "configuração fiscal não encontrada", err.Error()))
		return
	}

	// Atualizar certificado se fornecido
	if req.CertificateID != "" && req.CertificateID != existingConfig.CertificateID {
		existingConfig.CertificateID = req.CertificateID
	}

	// Atualizar configurações de NFe
	err = existingConfig.ConfigureNFe(
		req.NFeSeries,
		req.NFeNextNumber,
		req.NFeEnvironment,
		req.NFeCSCID,
		req.NFeCSCToken,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar NFe", err.Error()))
		return
	}

	// Atualizar configurações de NFCe
	err = existingConfig.ConfigureNFCe(
		req.NFCeSeries,
		req.NFCeNextNumber,
		req.NFCeEnvironment,
		req.NFCeCSCID,
		req.NFCeCSCToken,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar NFCe", err.Error()))
		return
	}

	// Atualizar configurações de SMTP
	err = existingConfig.ConfigureSMTP(
		req.SMTPHost,
		req.SMTPPort,
		req.SMTPUsername,
		req.SMTPPassword,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar SMTP", err.Error()))
		return
	}

	// Atualizar configurações de impressão
	existingConfig.ConfigurePrinting(
		req.PrintDANFEMode,
		req.PrinterName,
		req.PrinterPaperSize,
	)

	// Atualizar contingência
	if req.ContingencyEnabled {
		existingConfig.EnableContingency()
	} else {
		existingConfig.DisableContingency()
	}

	// Salvar alterações
	if err := c.fiscalRepo.Update(ctx, existingConfig); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar configuração fiscal", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewFiscalConfigResponse(existingConfig))
}

// @Summary Excluir configuração fiscal
// @Description Remove uma configuração fiscal pelo ID
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID da configuração fiscal"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/{id} [delete]
func (c *FiscalController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID da configuração fiscal não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Verificar se a configuração fiscal existe
	exists, err := c.fiscalRepo.Exists(ctx, id)
	if err != nil {
		c.logger.Error("erro ao verificar configuração fiscal", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao verificar configuração fiscal", err.Error()))
		return
	}

	if !exists {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "configuração fiscal não encontrada", fmt.Sprintf("configuração fiscal com ID %s não existe", id)))
		return
	}

	// Excluir a configuração fiscal
	if err := c.fiscalRepo.Delete(ctx, id); err != nil {
		c.logger.Error("erro ao excluir configuração fiscal", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao excluir configuração fiscal", err.Error()))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// @Summary Obter configuração fiscal por filial
// @Description Busca a configuração fiscal para uma filial específica
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param branch_id path string true "ID da filial"
// @Success 200 {object} dto.FiscalConfigResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/branch/{branch_id} [get]
func (c *FiscalController) GetByBranch(ctx *gin.Context) {
	branchID := ctx.Param("branch_id")
	if branchID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID da filial não fornecido"))
		return
	}

	// Buscar a configuração fiscal para a filial
	config, err := c.fiscalRepo.FindByBranch(ctx, branchID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "erro ao buscar configuração fiscal"
		if err.Error() == fmt.Sprintf("configuração fiscal para filial %s não encontrada", branchID) {
			statusCode = http.StatusNotFound
			errorMsg = "configuração fiscal não encontrada para esta filial"
		}
		ctx.JSON(statusCode, dto.NewErrorResponse(statusCode, errorMsg, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewFiscalConfigResponse(config))
}

// @Summary Incrementar numeração de NFe
// @Description Incrementa o próximo número de NFe para a filial
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param branch_id path string true "ID da filial"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/branch/{branch_id}/increment-nfe [post]
func (c *FiscalController) IncrementNFeNumber(ctx *gin.Context) {
	branchID := ctx.Param("branch_id")
	if branchID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "branch_id é obrigatório", ""))
		return
	}

	// Buscar configuração da filial
	config, err := c.fiscalRepo.FindByBranch(ctx, branchID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "configuração fiscal não encontrada", err.Error()))
		return
	}

	// Obter e incrementar número
	nextNumber := config.GetNextNFeNumber()

	// Atualizar no banco
	err = c.fiscalRepo.UpdateNFeNextNumber(ctx, config.ID, nextNumber)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar número de NFe", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"next_number": nextNumber})
}

// @Summary Incrementar numeração de NFCe
// @Description Incrementa o próximo número de NFCe para a filial
// @Tags Configurações Fiscais
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param branch_id path string true "ID da filial"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/branch/{branch_id}/increment-nfce [post]
func (c *FiscalController) IncrementNFCeNumber(ctx *gin.Context) {
	branchID := ctx.Param("branch_id")
	if branchID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "branch_id é obrigatório", ""))
		return
	}

	// Buscar configuração da filial
	config, err := c.fiscalRepo.FindByBranch(ctx, branchID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "configuração fiscal não encontrada", err.Error()))
		return
	}

	// Obter e incrementar número
	nextNumber := config.GetNextNFCeNumber()

	// Atualizar no banco
	err = c.fiscalRepo.UpdateNFCeNextNumber(ctx, config.ID, nextNumber)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar número de NFCe", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"next_number": nextNumber})
}

// @Summary Atualizar modo de contingência
// @Description Ativa ou desativa o modo de contingência para a filial
// @Tags Configurações Fiscais
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param branch_id path string true "ID da filial"
// @Param contingency body map[string]interface{} true "Dados de contingência"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /fiscal/configs/branch/{branch_id}/contingency [post]
func (c *FiscalController) UpdateContingency(ctx *gin.Context) {
	branchID := ctx.Param("branch_id")
	if branchID == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "branch_id é obrigatório", ""))
		return
	}

	// Buscar configuração da filial
	config, err := c.fiscalRepo.FindByBranch(ctx, branchID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "configuração fiscal não encontrada", err.Error()))
		return
	}

	// Atualizar modo de contingência
	if ctx.Query("enabled") == "true" {
		config.EnableContingency()
	} else {
		config.DisableContingency()
	}

	// Salvar alterações
	if err := c.fiscalRepo.Update(ctx, config); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar modo de contingência", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewFiscalConfigResponse(config))
}
