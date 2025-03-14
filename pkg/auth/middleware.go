package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/adapter/api/dto"
	"github.com/hugohenrick/erp-supermercado/pkg/tenant"
)

// JWTAuthMiddleware cria um middleware para autenticação JWT
func JWTAuthMiddleware() gin.HandlerFunc {
	jwtService, err := NewJWTService()
	if err != nil {
		// Se não conseguir criar o serviço JWT, retornar erro 500
		return func(c *gin.Context) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.NewErrorResponse(
				http.StatusInternalServerError,
				"Erro ao configurar autenticação",
				"O serviço JWT não foi inicializado corretamente",
			))
		}
	}

	return func(c *gin.Context) {
		// Obter o token do cabeçalho Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(
				http.StatusUnauthorized,
				"Autenticação requerida",
				"O cabeçalho Authorization não foi fornecido",
			))
			return
		}

		// Verificar o formato "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(
				http.StatusUnauthorized,
				"Formato de token inválido",
				"Use o formato 'Bearer <token>'",
			))
			return
		}

		// Validar o token
		claims, err := jwtService.ValidateToken(tokenParts[1])
		if err != nil {
			var statusCode int
			var message string

			if err == ErrExpiredToken {
				statusCode = http.StatusUnauthorized
				message = "Token expirado"
			} else {
				statusCode = http.StatusUnauthorized
				message = "Token inválido"
			}

			c.AbortWithStatusJSON(statusCode, dto.NewErrorResponse(
				statusCode,
				message,
				err.Error(),
			))
			return
		}

		// Armazenar as claims no contexto
		c.Set("user_id", claims.UserID)
		c.Set("tenant_id", claims.TenantID)
		c.Set("user_email", claims.Email)
		c.Set("user_name", claims.Name)
		c.Set("user_role", claims.Role)
		c.Set("branch_id", claims.BranchID)

		// Definir o tenant ID para o middleware de tenant
		c.Request = c.Request.WithContext(tenant.SetTenantIDContext(c.Request.Context(), claims.TenantID))

		c.Next()
	}
}

// RoleAuthMiddleware cria um middleware para verificação de papel/função do usuário
func RoleAuthMiddleware(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se o usuário está autenticado
		userRoleVal, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.NewErrorResponse(
				http.StatusUnauthorized,
				"Autenticação requerida",
				"",
			))
			return
		}

		// Verificar se o papel do usuário está na lista de papéis permitidos
		userRole, ok := userRoleVal.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, dto.NewErrorResponse(
				http.StatusInternalServerError,
				"Erro de tipo",
				"Falha ao obter o papel do usuário",
			))
			return
		}

		authorized := false
		for _, r := range roles {
			if userRole == r {
				authorized = true
				break
			}
		}

		if !authorized {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.NewErrorResponse(
				http.StatusForbidden,
				"Acesso negado",
				"Você não tem permissão para acessar este recurso",
			))
			return
		}

		c.Next()
	}
}

// GetCurrentUser obtém as informações do usuário atual do contexto
func GetCurrentUser(c *gin.Context) (string, string, string, string, string, string) {
	userID, _ := c.Get("user_id")
	tenantID, _ := c.Get("tenant_id")
	email, _ := c.Get("user_email")
	name, _ := c.Get("user_name")
	role, _ := c.Get("user_role")
	branchID, _ := c.Get("branch_id")

	userIDStr, _ := userID.(string)
	tenantIDStr, _ := tenantID.(string)
	emailStr, _ := email.(string)
	nameStr, _ := name.(string)
	roleStr, _ := role.(string)
	branchIDStr, _ := branchID.(string)

	return userIDStr, tenantIDStr, emailStr, nameStr, roleStr, branchIDStr
}
