package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
)

// SetupRoutes configura todas as rotas da API
func SetupRoutes(r *gin.Engine, handlers *handlers.Handlers) {
	// Configurar middleware, CORS, etc.

	// API versão 1
	v1 := r.Group("/api/v1")

	// Configurar rotas de autenticação
	SetupAuthRoutes(v1, handlers.AuthHandler)

	// Configurar rotas de usuários
	SetupUserRoutes(v1, handlers.UserHandler)

	// Configurar rotas de produtos
	SetupProductRoutes(v1, handlers.ProductHandler)

	// Configurar rotas de clientes
	SetupCustomerRoutes(v1, handlers.CustomerHandler)

	// Configurar rotas de MCP
	SetupMCPRoutes(v1, handlers.MCPHandler)
}
