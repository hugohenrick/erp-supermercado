package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/internal/domain/certificate"
	"github.com/hugohenrick/erp-supermercado/pkg/logger"
)

// CertificateController manipula as requisições relacionadas a certificados digitais
type CertificateController struct {
	certificateRepo certificate.Repository
	logger          logger.Logger
}

// NewCertificateController cria uma nova instância de CertificateController
func NewCertificateController(certificateRepo certificate.Repository, logger logger.Logger) *CertificateController {
	return &CertificateController{
		certificateRepo: certificateRepo,
		logger:          logger,
	}
}

// @Summary Criar certificado
// @Description Cria um novo certificado digital
// @Tags Certificados
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param certificate body dto.CertificateRequest true "Dados do certificado"
// @Success 201 {object} dto.CertificateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates [post]
func (c *CertificateController) Create(ctx *gin.Context) {
	var req dto.CertificateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	tenantID := ctx.GetString("tenant_id")

	// Verificar se pelo menos um dos dados do certificado foi fornecido
	if len(req.CertificateData) == 0 && req.CertificatePath == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados do certificado inválidos", "é necessário fornecer o arquivo do certificado ou o caminho para ele"))
		return
	}

	// Criar o certificado
	cert, err := certificate.NewCertificate(tenantID, req.BranchID, req.Name, req.ExpirationDate)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao criar certificado", err.Error()))
		return
	}

	// Armazenar os dados do certificado
	if len(req.CertificateData) > 0 {
		err = cert.StoreCertificateData(req.CertificateData, req.Password)
	} else {
		err = cert.StoreCertificatePath(req.CertificatePath, req.Password)
	}

	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar certificado", err.Error()))
		return
	}

	// Definir status do certificado
	if req.IsActive {
		cert.Activate()
	} else {
		cert.Deactivate()
	}

	// Salvar o certificado no repositório
	if err := c.certificateRepo.Create(ctx, cert); err != nil {
		c.logger.Error("erro ao salvar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao salvar certificado", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewCertificateResponse(cert))
}

// @Summary Upload de certificado
// @Description Realiza upload de um arquivo de certificado digital
// @Tags Certificados
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param file formData file true "Arquivo do certificado (.pfx)"
// @Param branch_id formData string true "ID da filial"
// @Param name formData string true "Nome do certificado"
// @Param password formData string true "Senha do certificado"
// @Param expiration_date formData string true "Data de validade (YYYY-MM-DD)"
// @Param is_active formData boolean false "Se o certificado está ativo"
// @Success 201 {object} dto.CertificateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/upload [post]
func (c *CertificateController) Upload(ctx *gin.Context) {
	// Obter os dados do formulário
	branchID := ctx.PostForm("branch_id")
	name := ctx.PostForm("name")
	password := ctx.PostForm("password")
	expirationDateStr := ctx.PostForm("expiration_date")
	isActiveStr := ctx.PostForm("is_active")

	// Validar campos obrigatórios
	if branchID == "" || name == "" || password == "" || expirationDateStr == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", "todos os campos são obrigatórios"))
		return
	}

	// Parsear a data de validade
	expirationDate, err := time.Parse("2006-01-02", expirationDateStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "data inválida", "formato esperado: YYYY-MM-DD"))
		return
	}

	// Verificar se o certificado está ativo
	isActive := false
	if isActiveStr != "" {
		isActive, err = strconv.ParseBool(isActiveStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "valor inválido", "is_active deve ser true ou false"))
			return
		}
	}

	// Obter o arquivo do certificado
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "arquivo inválido", err.Error()))
		return
	}

	// Validar o tipo do arquivo
	if file.Size <= 0 {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "arquivo inválido", "arquivo vazio"))
		return
	}

	// Abrir o arquivo para leitura
	src, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao ler arquivo", err.Error()))
		return
	}
	defer src.Close()

	// Ler o conteúdo do arquivo
	buffer := bytes.NewBuffer(nil)
	if _, err := io.Copy(buffer, src); err != nil {
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao ler arquivo", err.Error()))
		return
	}

	tenantID := ctx.GetString("tenant_id")

	// Criar o certificado
	cert, err := certificate.NewCertificate(tenantID, branchID, name, expirationDate)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao criar certificado", err.Error()))
		return
	}

	// Armazenar os dados do certificado
	if err := cert.StoreCertificateData(buffer.Bytes(), password); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao configurar certificado", err.Error()))
		return
	}

	// Definir status do certificado
	if isActive {
		cert.Activate()
	} else {
		cert.Deactivate()
	}

	// Salvar o certificado no repositório
	if err := c.certificateRepo.Create(ctx, cert); err != nil {
		c.logger.Error("erro ao salvar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao salvar certificado", err.Error()))
		return
	}

	ctx.JSON(http.StatusCreated, dto.NewCertificateResponse(cert))
}

// @Summary Obter certificado
// @Description Busca um certificado pelo ID
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do certificado"
// @Success 200 {object} dto.CertificateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/{id} [get]
func (c *CertificateController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID do certificado não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Buscar o certificado no repositório
	cert, err := c.certificateRepo.FindByID(ctx, id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "erro ao buscar certificado"
		if err.Error() == fmt.Sprintf("certificado com ID %s não encontrado", id) {
			statusCode = http.StatusNotFound
			errorMsg = "certificado não encontrado"
		}
		ctx.JSON(statusCode, dto.NewErrorResponse(statusCode, errorMsg, err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewCertificateResponse(cert))
}

// @Summary Listar certificados
// @Description Lista os certificados com paginação
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param page query int false "Número da página (padrão: 1)"
// @Param page_size query int false "Tamanho da página (padrão: 10)"
// @Param branch_id query string false "Filtrar por filial"
// @Success 200 {object} dto.CertificateListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates [get]
func (c *CertificateController) List(ctx *gin.Context) {
	// Parâmetros de paginação
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "10"))

	// Validar página e tamanho
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Calcular offset
	offset := (page - 1) * pageSize

	// Verificar se há filtro por filial
	branchID := ctx.Query("branch_id")

	var certificates []*certificate.Certificate
	var err error
	var total int

	// Recuperar o tenant ID do contexto
	tenantID := ctx.GetString("tenant_id")

	// Buscar certificados pelo tenant ou filial
	if branchID != "" {
		certificates, err = c.certificateRepo.FindByBranch(ctx, branchID)
		if err != nil {
			c.logger.Error("erro ao listar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar certificados", err.Error()))
			return
		}

		// Contar total de certificados para a filial
		total, err = c.certificateRepo.CountByBranch(ctx, branchID)
		if err != nil {
			c.logger.Error("erro ao contar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao contar certificados", err.Error()))
			return
		}
	} else {
		// Listar todos os certificados do tenant
		certificates, err = c.certificateRepo.List(ctx, tenantID, pageSize, offset)
		if err != nil {
			c.logger.Error("erro ao listar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar certificados", err.Error()))
			return
		}

		// Contar total de certificados para o tenant
		total, err = c.certificateRepo.CountByTenant(ctx, tenantID)
		if err != nil {
			c.logger.Error("erro ao contar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao contar certificados", err.Error()))
			return
		}
	}

	// Retornar a lista de certificados
	ctx.JSON(http.StatusOK, dto.NewCertificateListResponse(certificates, total, page, pageSize))
}

// @Summary Atualizar certificado
// @Description Atualiza um certificado existente
// @Tags Certificados
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do certificado"
// @Param certificate body dto.CertificateRequest true "Dados do certificado"
// @Success 200 {object} dto.CertificateResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/{id} [put]
func (c *CertificateController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID do certificado não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Obter os dados do certificado da requisição
	var req dto.CertificateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "dados inválidos", err.Error()))
		return
	}

	// Buscar o certificado existente
	existingCert, err := c.certificateRepo.FindByID(ctx, id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "erro ao buscar certificado"
		if err.Error() == fmt.Sprintf("certificado com ID %s não encontrado", id) {
			statusCode = http.StatusNotFound
			errorMsg = "certificado não encontrado"
		}
		ctx.JSON(statusCode, dto.NewErrorResponse(statusCode, errorMsg, err.Error()))
		return
	}

	// Atualizar os dados do certificado
	if req.Name != "" {
		existingCert.Name = req.Name
	}

	// Atualizar dados do certificado se fornecidos
	if len(req.CertificateData) > 0 && req.Password != "" {
		if err := existingCert.StoreCertificateData(req.CertificateData, req.Password); err != nil {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao atualizar dados do certificado", err.Error()))
			return
		}
	} else if req.CertificatePath != "" && req.Password != "" {
		if err := existingCert.StoreCertificatePath(req.CertificatePath, req.Password); err != nil {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao atualizar caminho do certificado", err.Error()))
			return
		}
	}

	// Atualizar data de validade se fornecida
	if !req.ExpirationDate.IsZero() && req.ExpirationDate != existingCert.ExpirationDate {
		if err := existingCert.RenewExpiration(req.ExpirationDate); err != nil {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao atualizar data de validade", err.Error()))
			return
		}
	}

	// Atualizar status do certificado
	if req.IsActive && !existingCert.IsActive {
		existingCert.Activate()
	} else if !req.IsActive && existingCert.IsActive {
		existingCert.Deactivate()
	}

	// Salvar as alterações no certificado
	if err := c.certificateRepo.Update(ctx, existingCert); err != nil {
		c.logger.Error("erro ao atualizar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao atualizar certificado", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewCertificateResponse(existingCert))
}

// @Summary Excluir certificado
// @Description Remove um certificado pelo ID
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do certificado"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/{id} [delete]
func (c *CertificateController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID do certificado não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Verificar se o certificado existe
	exists, err := c.certificateRepo.Exists(ctx, id)
	if err != nil {
		c.logger.Error("erro ao verificar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao verificar certificado", err.Error()))
		return
	}

	if !exists {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "certificado não encontrado", fmt.Sprintf("certificado com ID %s não existe", id)))
		return
	}

	// Excluir o certificado
	if err := c.certificateRepo.Delete(ctx, id); err != nil {
		c.logger.Error("erro ao excluir certificado", "error", err.Error())
		if err.Error() == fmt.Sprintf("não é possível excluir o certificado pois está em uso em configurações fiscais") {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "certificado em uso", err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao excluir certificado", err.Error()))
		return
	}

	ctx.Status(http.StatusNoContent)
}

// @Summary Ativar certificado
// @Description Ativa um certificado pelo ID (e desativa outros certificados da mesma filial)
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do certificado"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/{id}/activate [post]
func (c *CertificateController) Activate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID do certificado não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Verificar se o certificado existe
	exists, err := c.certificateRepo.Exists(ctx, id)
	if err != nil {
		c.logger.Error("erro ao verificar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao verificar certificado", err.Error()))
		return
	}

	if !exists {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "certificado não encontrado", fmt.Sprintf("certificado com ID %s não existe", id)))
		return
	}

	// Ativar o certificado
	if err := c.certificateRepo.Activate(ctx, id); err != nil {
		c.logger.Error("erro ao ativar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao ativar certificado", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("certificado ativado com sucesso", nil))
}

// @Summary Desativar certificado
// @Description Desativa um certificado pelo ID
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "ID do certificado"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/{id}/deactivate [post]
func (c *CertificateController) Deactivate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "ID do certificado não fornecido"))
		return
	}

	// Validar formato do ID
	if _, err := uuid.Parse(id); err != nil {
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "ID inválido", "formato de ID inválido"))
		return
	}

	// Verificar se o certificado existe
	exists, err := c.certificateRepo.Exists(ctx, id)
	if err != nil {
		c.logger.Error("erro ao verificar certificado", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao verificar certificado", err.Error()))
		return
	}

	if !exists {
		ctx.JSON(http.StatusNotFound, dto.NewErrorResponse(http.StatusNotFound, "certificado não encontrado", fmt.Sprintf("certificado com ID %s não existe", id)))
		return
	}

	// Desativar o certificado
	if err := c.certificateRepo.Deactivate(ctx, id); err != nil {
		c.logger.Error("erro ao desativar certificado", "error", err.Error())
		if err.Error() == fmt.Sprintf("não é possível desativar o certificado pois está em uso em configurações fiscais") {
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "certificado em uso", err.Error()))
			return
		}
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao desativar certificado", err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, dto.NewSuccessResponse("certificado desativado com sucesso", nil))
}

// @Summary Listar certificados expirando
// @Description Lista os certificados que expirarão em X dias
// @Tags Certificados
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param days query int false "Número de dias (padrão: 30)"
// @Success 200 {object} dto.CertificateListResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /certificates/expiring [get]
func (c *CertificateController) ListExpiring(ctx *gin.Context) {
	// Obter o número de dias
	days, _ := strconv.Atoi(ctx.DefaultQuery("days", "30"))
	if days < 1 {
		days = 30
	}

	// Buscar certificados que expirarão em X dias
	certificates, err := c.certificateRepo.FindExpiring(ctx, days)
	if err != nil {
		c.logger.Error("erro ao listar certificados expirando", "error", err.Error())
		ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar certificados expirando", err.Error()))
		return
	}

	// Retornar a lista de certificados
	ctx.JSON(http.StatusOK, dto.NewCertificateListResponse(certificates, len(certificates), 1, len(certificates)))
}
