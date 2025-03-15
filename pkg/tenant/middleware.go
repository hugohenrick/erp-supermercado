package tenant

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
)

// TenantValidator define a interface para validação de tenant
type TenantValidator interface {
	ValidateTenant(tenantID string) (bool, error)
}

// TenantMiddleware cria um middleware para validação do tenant
func TenantMiddleware(validator TenantValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Não aplicar a validação para rotas públicas
		if isExcludedPath(c.FullPath()) {
			c.Next()
			return
		}

		// Obter tenant ID do cabeçalho
		tenantID := c.GetHeader("tenant-id")
		if tenantID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, dto.NewErrorResponse(
				http.StatusBadRequest,
				"Tenant ID não fornecido",
				"O cabeçalho 'tenant-id' é obrigatório",
			))
			return
		}

		// Validar o tenant ID
		valid, err := validator.ValidateTenant(tenantID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.NewErrorResponse(
				http.StatusInternalServerError,
				"Erro ao validar tenant",
				err.Error(),
			))
			return
		}

		if !valid {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.NewErrorResponse(
				http.StatusForbidden,
				"Tenant inválido",
				"O tenant informado não existe ou está inativo",
			))
			return
		}

		// Armazenar o tenant ID no contexto
		c.Set("tenant_id", tenantID)
		c.Request = c.Request.WithContext(SetTenantIDContext(c.Request.Context(), tenantID))

		c.Next()
	}
}

// isExcludedPath verifica se o caminho está excluído da validação de tenant
func isExcludedPath(path string) bool {
	// Lista de caminhos excluídos da validação de tenant
	excludedPaths := []string{
		"/api/v1/auth/login",
		"/api/v1/tenants",
		"/api/v1/tenants/",
		"/api/v1/health",
		"/api/v1/setup/admin", // Rota para criar o primeiro usuário administrador
	}

	for _, excludedPath := range excludedPaths {
		if path == excludedPath {
			return true
		}
	}

	return false
}
