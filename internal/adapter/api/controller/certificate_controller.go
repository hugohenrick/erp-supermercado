package controller

import (
	"bytes"
	"crypto/x509"
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
	"github.com/hugohenrick/erp-supermercado/pkg/pkcs12"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
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

// getTenantID é um método auxiliar que tenta obter o tenant ID de várias fontes
func (c *CertificateController) getTenantID(ctx *gin.Context) string {
	// Primeiro tenta obter do contexto
	tenantID := ctx.GetString("tenant_id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in context", "tenant_id", tenantID)
		return tenantID
	}

	// Tenta vários formatos possíveis de headers
	tenantID = ctx.GetHeader("Tenant-Id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in Tenant-Id header", "tenant_id", tenantID)
		return tenantID
	}

	tenantID = ctx.GetHeader("tenant-id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in tenant-id header", "tenant_id", tenantID)
		return tenantID
	}

	tenantID = ctx.GetHeader("tenant_id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in tenant_id header", "tenant_id", tenantID)
		return tenantID
	}

	tenantID = ctx.GetHeader("X-Tenant-Id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in X-Tenant-Id header", "tenant_id", tenantID)
		return tenantID
	}

	tenantID = ctx.GetHeader("x-tenant-id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in x-tenant-id header", "tenant_id", tenantID)
		return tenantID
	}

	// Por último, tenta pegar do formulário
	tenantID = ctx.PostForm("tenant_id")
	if tenantID != "" {
		c.logger.Info("Tenant ID found in form data", "tenant_id", tenantID)
	} else {
		c.logger.Warn("Tenant ID not found in any source (context, headers or form)")
	}
	return tenantID
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
	c.logger.Info("Starting certificate creation")

	// Log content type and headers
	c.logger.Info("Content-Type:", "header", ctx.GetHeader("Content-Type"))
	c.logger.Info("Request Headers:", "headers", ctx.Request.Header)

	// Log request body parsing attempt
	var req dto.CertificateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		c.logger.Error("Failed to bind JSON request", "error", err.Error())
		c.logger.Info("Attempting to parse as multipart form")

		// Try to parse as multipart form
		if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
			c.logger.Error("Failed to parse multipart form", "error", err)
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao processar formulário", err.Error()))
			return
		}

		// Log form fields
		c.logger.Info("Form fields received:", "fields", ctx.Request.PostForm)
		c.logger.Info("File headers received:", "files", ctx.Request.MultipartForm.File)

		// Get form values
		req.Name = ctx.PostForm("name")
		req.BranchID = ctx.PostForm("branch_id")
		req.Password = ctx.PostForm("password")
		expirationDateStr := ctx.PostForm("expiration_date")
		isActiveStr := ctx.PostForm("is_active")

		c.logger.Info("Parsed form values",
			"name", req.Name,
			"branch_id", req.BranchID,
			"password_length", len(req.Password),
			"expiration_date", expirationDateStr,
			"is_active", isActiveStr)

		// Parse expiration date
		expDate, err := time.Parse("20060102150405", expirationDateStr)
		if err != nil {
			c.logger.Error("Failed to parse expiration date", "error", err, "value", expirationDateStr)
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "data de expiração inválida", "formato esperado: AAAAMMDDHHMMSS"))
			return
		}
		req.ExpirationDate = expDate

		// Parse is_active
		if isActiveStr != "" {
			isActive, err := strconv.ParseBool(isActiveStr)
			if err != nil {
				c.logger.Error("Failed to parse is_active", "error", err)
				ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "valor inválido para is_active", "deve ser true ou false"))
				return
			}
			req.IsActive = isActive
		}

		// Get certificate file
		file, err := ctx.FormFile("certificate")
		if err != nil {
			c.logger.Error("Failed to get certificate file", "error", err)
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "arquivo do certificado não fornecido", err.Error()))
			return
		}

		// Open and read certificate file
		certFile, err := file.Open()
		if err != nil {
			c.logger.Error("Failed to open certificate file", "error", err)
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao abrir arquivo do certificado", err.Error()))
			return
		}
		defer certFile.Close()

		certData, err := io.ReadAll(certFile)
		if err != nil {
			c.logger.Error("Failed to read certificate file", "error", err)
			ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "erro ao ler arquivo do certificado", err.Error()))
			return
		}
		req.CertificateData = certData
	}

	c.logger.Info("Request data validated successfully")

	tenantID := c.getTenantID(ctx)
	// Verifica se o tenant_id ainda está vazio
	if tenantID == "" {
		c.logger.Error("tenant ID not found in context or headers")
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "tenant ID não encontrado", "tenant ID não encontrado no contexto ou headers"))
		return
	}

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
	// Obter o contexto com o tenant ID definido
	reqCtx := ctx.Request.Context()
	reqCtx = tenant.SetTenantIDContext(reqCtx, tenantID)

	if err := c.certificateRepo.Create(reqCtx, cert); err != nil {
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

	tenantID := c.getTenantID(ctx)
	// Verifica se o tenant_id ainda está vazio
	if tenantID == "" {
		c.logger.Error("tenant ID not found in context or headers")
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "tenant ID não encontrado", "tenant ID não encontrado no contexto ou headers"))
		return
	}

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
	// Obter o contexto com o tenant ID definido
	reqCtx := ctx.Request.Context()
	reqCtx = tenant.SetTenantIDContext(reqCtx, tenantID)

	if err := c.certificateRepo.Create(reqCtx, cert); err != nil {
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

	// Obter o tenant ID
	tenantID := c.getTenantID(ctx)
	if tenantID == "" {
		c.logger.Error("tenant ID not found in context or headers")
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "tenant ID não encontrado", "tenant ID não encontrado no contexto ou headers"))
		return
	}

	// Buscar o certificado no repositório
	// Obter o contexto com o tenant ID definido
	reqCtx := ctx.Request.Context()
	reqCtx = tenant.SetTenantIDContext(reqCtx, tenantID)

	cert, err := c.certificateRepo.FindByID(reqCtx, id)
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
	tenantID := c.getTenantID(ctx)
	// Verifica se o tenant_id ainda está vazio
	if tenantID == "" {
		c.logger.Error("tenant ID not found in context or headers for List method")
		ctx.JSON(http.StatusBadRequest, dto.NewErrorResponse(http.StatusBadRequest, "tenant ID não encontrado", "tenant ID não encontrado no contexto ou headers"))
		return
	}

	// Buscar certificados pelo tenant ou filial
	// Obter o contexto com o tenant ID definido
	reqCtx := ctx.Request.Context()
	reqCtx = tenant.SetTenantIDContext(reqCtx, tenantID)

	if branchID != "" {
		certificates, err = c.certificateRepo.FindByBranch(reqCtx, branchID)
		if err != nil {
			c.logger.Error("erro ao listar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar certificados", err.Error()))
			return
		}

		// Contar total de certificados para a filial
		total, err = c.certificateRepo.CountByBranch(reqCtx, branchID)
		if err != nil {
			c.logger.Error("erro ao contar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao contar certificados", err.Error()))
			return
		}
	} else {
		// Listar todos os certificados do tenant
		certificates, err = c.certificateRepo.List(reqCtx, tenantID, pageSize, offset)
		if err != nil {
			c.logger.Error("erro ao listar certificados", "error", err.Error())
			ctx.JSON(http.StatusInternalServerError, dto.NewErrorResponse(http.StatusInternalServerError, "erro ao listar certificados", err.Error()))
			return
		}

		// Contar total de certificados para o tenant
		total, err = c.certificateRepo.CountByTenant(reqCtx, tenantID)
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

// ExtractInfo extrai informações de um certificado digital
func (c *CertificateController) ExtractInfo(ctx *gin.Context) {
	c.logger.Info("Starting certificate info extraction")

	// Log form content type and all headers
	c.logger.Info("Content-Type header: " + ctx.GetHeader("Content-Type"))
	c.logger.Info("All Headers:", "headers", ctx.Request.Header)

	// Log request method and URL
	c.logger.Info("Request details",
		"method", ctx.Request.Method,
		"url", ctx.Request.URL.String(),
		"content_length", ctx.Request.ContentLength)

	// Log all form fields
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.logger.Error("Failed to parse multipart form", "error", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "erro ao processar formulário",
			"details": err.Error(),
		})
		return
	}

	// Log form details
	if ctx.Request.MultipartForm != nil {
		c.logger.Info("Form fields found:", "field_names", ctx.Request.MultipartForm.Value)
		c.logger.Info("File fields found:", "file_names", ctx.Request.MultipartForm.File)

		if files, exists := ctx.Request.MultipartForm.File["certificate"]; exists {
			c.logger.Info("Certificate file details",
				"count", len(files),
				"first_file", map[string]interface{}{
					"filename": files[0].Filename,
					"size":     files[0].Size,
					"header":   files[0].Header,
				})
		} else {
			c.logger.Error("No certificate file field found in form")
		}
	} else {
		c.logger.Error("MultipartForm is nil after parsing")
	}

	// Get the certificate file from the request
	file, err := ctx.FormFile("certificate")
	if err != nil {
		c.logger.Error("Failed to get certificate file from request", "error", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "arquivo não fornecido",
			"details": err.Error(),
		})
		return
	}

	// Log file details
	c.logger.Info("Certificate file received",
		"filename", file.Filename,
		"size", file.Size,
		"header", file.Header)

	// Get password from form
	password := ctx.PostForm("password")
	if password == "" {
		c.logger.Error("Password not provided in request")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "senha não fornecida",
			"details": "password field is required",
		})
		return
	}

	// Open the uploaded file
	certFile, err := file.Open()
	if err != nil {
		c.logger.Error("Failed to open certificate file", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "erro ao abrir arquivo",
			"details": err.Error(),
		})
		return
	}
	defer certFile.Close()

	// Read the file content
	certData, err := io.ReadAll(certFile)
	if err != nil {
		c.logger.Error("Failed to read certificate file", "error", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "erro ao ler arquivo",
			"details": err.Error(),
		})
		return
	}

	c.logger.Info("Successfully read certificate file", "size", len(certData))

	// Convert PKCS12 to PEM
	blocks, err := pkcs12.ToPEM(certData, password)
	if err != nil {
		c.logger.Error("Failed to convert PKCS12 to PEM", "error", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "erro ao processar certificado",
			"details": err.Error(),
		})
		return
	}

	c.logger.Info("Successfully converted PKCS12 to PEM", "blocks_count", len(blocks))

	// Find the certificate block
	var cert *x509.Certificate
	for _, block := range blocks {
		if block.Type == "CERTIFICATE" {
			cert, err = x509.ParseCertificate(block.Bytes)
			if err != nil {
				c.logger.Error("Failed to parse certificate", "error", err)
				continue
			}
			// Use the first valid certificate found
			break
		}
	}

	if cert == nil {
		c.logger.Error("No valid certificate found in PKCS12 file")
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "certificado inválido",
			"details": "no valid certificate found in file",
		})
		return
	}

	c.logger.Info("Successfully extracted certificate info",
		"subject", cert.Subject.String(),
		"expiration", cert.NotAfter)

	// Recupera o tenant ID para logs
	tenantID := c.getTenantID(ctx)
	if tenantID == "" {
		c.logger.Info("No tenant ID found for logging in ExtractInfo")
	} else {
		c.logger.Info("Tenant ID for logging in ExtractInfo", "tenant_id", tenantID)

		// Set tenant ID in request context for potential future uses
		reqCtx := ctx.Request.Context()
		reqCtx = tenant.SetTenantIDContext(reqCtx, tenantID)
		ctx.Request = ctx.Request.WithContext(reqCtx)
	}

	// Return the certificate information
	ctx.JSON(http.StatusOK, dto.CertificateExtractResponse{
		ExpirationDate: cert.NotAfter,
		Subject:        cert.Subject.String(),
	})
}
