package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/hugohenrick/erp-supermercado/internal/api/handlers"
	"github.com/hugohenrick/erp-supermercado/pkg/auth"
)

// SetupMCPRoutes configura as rotas para o MCP
func SetupMCPRoutes(router *gin.RouterGroup, mcpHandler *handlers.MCPHandler) {
	mcpRouter := router.Group("/mcp")
	mcpRouter.Use(auth.JWTAuthMiddleware())
	{
		// Processar uma mensagem
		mcpRouter.POST("/message", mcpHandler.ProcessMessage)

		// Limpar o histórico de mensagens
		mcpRouter.DELETE("/history", mcpHandler.ClearHistory)

		// Obter histórico de mensagens
		mcpRouter.GET("/history", mcpHandler.GetHistoryMessages)
	}
}
