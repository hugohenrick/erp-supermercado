package tenant

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Chaves utilizadas no contexto
type contextKey string

const (
	// TenantIDKey é a chave para o ID do tenant no contexto
	TenantIDKey contextKey = "tenant_id"

	// BranchIDKey é a chave para o ID da filial no contexto
	BranchIDKey contextKey = "branch_id"
)

var (
	// ErrTenantNotSpecified é retornado quando o tenant não é especificado na requisição
	ErrTenantNotSpecified = errors.New("tenant não especificado na requisição")

	// ErrTenantNotFound é retornado quando o tenant não é encontrado
	ErrTenantNotFound = errors.New("tenant não encontrado")

	// ErrTenantNotActive é retornado quando o tenant não está ativo
	ErrTenantNotActive = errors.New("tenant não está ativo")
)

// TenantExtractor é a interface para extrair o ID do tenant da requisição
type TenantExtractor interface {
	// Extract extrai o ID do tenant da requisição
	Extract(c *gin.Context) (string, error)
}

// HeaderTenantExtractor extrai o tenant do cabeçalho da requisição
type HeaderTenantExtractor struct {
	headerName string
}

// NewHeaderTenantExtractor cria um novo extrator de tenant baseado em cabeçalho
func NewHeaderTenantExtractor(headerName string) *HeaderTenantExtractor {
	if headerName == "" {
		headerName = os.Getenv("TENANT_HEADER")
		if headerName == "" {
			headerName = "X-Tenant-ID"
		}
	}
	return &HeaderTenantExtractor{headerName: headerName}
}

// Extract implementa TenantExtractor.Extract
func (e *HeaderTenantExtractor) Extract(c *gin.Context) (string, error) {
	tenantID := c.GetHeader(e.headerName)
	if tenantID == "" {
		// Tenta obter de parâmetros de URL ou query
		tenantID = c.Param("tenant_id")
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}
	}

	if tenantID == "" {
		// Usa tenant padrão se configurado
		tenantID = os.Getenv("DEFAULT_TENANT")
		if tenantID == "" {
			return "", ErrTenantNotSpecified
		}
	}

	return tenantID, nil
}

// Middleware cria um middleware Gin para resolução de tenant
func Middleware(extractor TenantExtractor, validator TenantValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Ignora rotas públicas
		if isPublicRoute(c.Request.URL.Path) {
			c.Next()
			return
		}

		tenantID, err := extractor.Extract(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		// Valida o tenant se um validador foi fornecido
		if validator != nil {
			if err := validator.Validate(c, tenantID); err != nil {
				if errors.Is(err, ErrTenantNotFound) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error": "tenant não encontrado",
					})
					return
				}
				if errors.Is(err, ErrTenantNotActive) {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": "tenant não está ativo",
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "erro ao validar tenant",
				})
				return
			}
		}

		// Adiciona o tenant ID ao contexto
		ctx := context.WithValue(c.Request.Context(), TenantIDKey, tenantID)

		// Tenta extrair o ID da filial, se estiver presente
		branchID := c.GetHeader("X-Branch-ID")
		if branchID == "" {
			branchID = c.Param("branch_id")
			if branchID == "" {
				branchID = c.Query("branch_id")
			}
		}

		if branchID != "" {
			ctx = context.WithValue(ctx, BranchIDKey, branchID)
		}

		// Atualiza o contexto da requisição
		c.Request = c.Request.WithContext(ctx)

		// Define headers para resposta
		c.Writer.Header().Set(extractor.(*HeaderTenantExtractor).headerName, tenantID)

		c.Next()
	}
}

// TenantValidator valida se um tenant existe e está ativo
type TenantValidator interface {
	// Validate valida se um tenant existe e está ativo
	Validate(c *gin.Context, tenantID string) error
}

// GetTenantID retorna o ID do tenant do contexto
func GetTenantID(ctx context.Context) string {
	value := ctx.Value(TenantIDKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

// GetBranchID retorna o ID da filial do contexto
func GetBranchID(ctx context.Context) string {
	value := ctx.Value(BranchIDKey)
	if value == nil {
		return ""
	}
	return value.(string)
}

// isPublicRoute verifica se a rota é pública (não requer tenant)
func isPublicRoute(path string) bool {
	publicPaths := []string{
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}
