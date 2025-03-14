package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/pkg/jwt"
)

// AuthMiddleware é o middleware para autenticação
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "token não informado", ""))
			return
		}

		// Verificar se o header começa com "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "token inválido", ""))
			return
		}

		// Extrair o token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validar o token
		claims, err := jwt.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(http.StatusUnauthorized, "token inválido", err.Error()))
			return
		}

		// Adicionar as claims ao contexto
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("branch_id", claims.BranchID)

		c.Next()
	}
}
